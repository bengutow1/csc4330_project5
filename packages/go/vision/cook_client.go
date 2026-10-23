// Author: LAPTOP-BPMQJP10\andyk
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
)

// Recipes sends only foods and the kitchen settings that constrain a dish. The handler still
// validates every recipe and applies recipes.Check before it can reach the player.
func (c *Client) Recipes(ctx context.Context, items []Item, kitchen Kitchen) ([]Recipe, error) {
	const op = "vision: recipes"
	input := struct {
		Ingredients []wireItem `json:"ingredients"`
		Staples     []string   `json:"staples"`
		Equipment   []string   `json:"equipment"`
		Diet        string     `json:"diet"`
		Allergies   []string   `json:"allergies"`
	}{Ingredients: make([]wireItem, 0, len(items)), Staples: kitchen.Staples, Equipment: kitchen.Equipment, Diet: kitchen.Diet, Allergies: kitchen.Allergies}
	for _, item := range items {
		input.Ingredients = append(input.Ingredients, wireItem(item))
	}
	contextJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("%s: encode context: %w", op, err)
	}
	text, err := c.structured(ctx, op, recipesPrompt, recipesSchema, []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock(string(contextJSON))})
	if err != nil {
		return nil, err
	}
	var body struct {
		Recipes *[]wireRecipe `json:"recipes"`
	}
	err = decodeCookAnswer(text, &body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %w", op, err, ErrUnavailable)
	}
	if body.Recipes == nil || len(*body.Recipes) > 12 {
		return nil, fmt.Errorf("%s: missing or oversized recipe list: %w", op, ErrUnavailable)
	}
	out := make([]Recipe, 0, len(*body.Recipes))
	for _, r := range *body.Recipes {
		if r.Title == nil || r.Minutes == nil || r.Difficulty == nil || r.Equipment == nil || r.Uses == nil || r.Missing == nil || r.Allergens == nil || r.Steps == nil || r.StepTimers == nil {
			return nil, fmt.Errorf("%s: incomplete recipe: %w", op, ErrUnavailable)
		}
		if len(*r.StepTimers) != len(*r.Steps) {
			return nil, fmt.Errorf("%s: timers do not match steps: %w", op, ErrUnavailable)
		}
		out = append(out, Recipe{Title: *r.Title, Minutes: *r.Minutes, Difficulty: *r.Difficulty, Equipment: *r.Equipment, Uses: *r.Uses, Missing: *r.Missing, Allergens: *r.Allergens, Steps: *r.Steps, StepTimers: *r.StepTimers})
	}
	return out, nil
}

type wireRecipe struct {
	Title      *string   `json:"title"`
	Minutes    *int      `json:"minutes"`
	Difficulty *string   `json:"difficulty"`
	Equipment  *[]string `json:"equipment"`
	Uses       *[]string `json:"uses"`
	Missing    *[]string `json:"missing"`
	Allergens  *[]string `json:"allergens"`
	Steps      *[]string `json:"steps"`
	StepTimers *[]int    `json:"step_timers"`
}

// CheckPlate passes only when both observations are true; it never assigns points.
func (c *Client) CheckPlate(ctx context.Context, photo []byte, recipeTitle string) (PlateCheck, error) {
	const op = "vision: check plate"
	title, err := json.Marshal(struct {
		Title string `json:"recipe_title"`
	}{recipeTitle})
	if err != nil {
		return PlateCheck{}, fmt.Errorf("%s: encode title: %w", op, err)
	}
	content := []anthropic.BetaContentBlockParamUnion{
		anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{Data: base64.StdEncoding.EncodeToString(photo), MediaType: anthropic.BetaBase64ImageSourceMediaTypeImageJPEG}),
		anthropic.NewBetaTextBlock(string(title)),
	}
	text, err := c.structured(ctx, op, platePrompt, plateSchema, content)
	if err != nil {
		return PlateCheck{}, err
	}
	var body struct {
		Food    *bool   `json:"shows_food"`
		Matches *bool   `json:"matches_recipe"`
		Reason  *string `json:"reason"`
	}
	err = decodeCookAnswer(text, &body)
	if err != nil {
		return PlateCheck{}, fmt.Errorf("%s: %w: %w", op, err, ErrUnavailable)
	}
	if body.Food == nil || body.Matches == nil || body.Reason == nil {
		return PlateCheck{}, fmt.Errorf("%s: incomplete plate check: %w", op, ErrUnavailable)
	}
	reason := strings.TrimSpace(*body.Reason)
	if reason == "" || strings.ContainsAny(reason, "\r\n\x00") || utf8.RuneCountInString(reason) > 200 {
		return PlateCheck{}, fmt.Errorf("%s: reason must be one short line: %w", op, ErrUnavailable)
	}
	return PlateCheck{Passed: *body.Food && *body.Matches, Reason: reason}, nil
}

// structured keeps all three vision operations on the same no-retry request and error path.
func (c *Client) structured(ctx context.Context, op, prompt string, schema map[string]any, content []anthropic.BetaContentBlockParamUnion) (string, error) {
	msg, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model: c.model, MaxTokens: readMaxTokens, System: []anthropic.BetaTextBlockParam{{Text: prompt}},
		Messages:     []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(content...)},
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortLow, Format: anthropic.BetaJSONOutputFormatParam{Schema: schema}},
		Fallbacks:    anthropic.BetaFallbacksParamOfDefault(), Betas: []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return "", c.fail(ctx, op, err)
	}
	if msg.StopReason != anthropic.BetaStopReasonEndTurn {
		return "", fmt.Errorf("%s: incomplete model answer: %w", op, ErrUnavailable)
	}
	var text strings.Builder
	for _, block := range msg.Content {
		b, ok := block.AsAny().(anthropic.BetaTextBlock)
		if ok {
			text.WriteString(b.Text)
		}
	}
	return text.String(), nil
}

func decodeCookAnswer(text string, out any) error {
	dec := json.NewDecoder(bytes.NewReader([]byte(text)))
	dec.DisallowUnknownFields()
	err := dec.Decode(out)
	if err != nil {
		return fmt.Errorf("decode answer: %w", err)
	}
	err = dec.Decode(new(any))
	if !errors.Is(err, io.EOF) {
		return errors.New("decode answer: data after the object")
	}
	return nil
}
