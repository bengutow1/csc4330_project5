// Author: Ben Gutowski
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// DefaultModel is the Claude model a fridge read uses when main.go names none.
const DefaultModel = "claude-opus-5-5"

// readTimeout caps one fridge read: up to three photos usually take several seconds, and the
// cap stays well under the API's 90 s write timeout so a hung call cannot hold the request open.
const readTimeout = 45 * time.Second

// readMaxTokens leaves the model room to think before a short JSON answer, without risking a
// cut-off that would fail to decode.
const readMaxTokens = 16000

// Config is what Client needs to reach the model.
type Config struct {
	// APIKey is the Anthropic key, from VISION_API_KEY.
	APIKey string
	// Model is the Claude model id. "" uses DefaultModel.
	Model string
	// BaseURL replaces Anthropic's API address, for tests. "" is the real API.
	BaseURL string
}

// Client reads kitchens with Claude through Anthropic's Messages API. It never retries: the
// player taps again, and a retry loop would multiply the bill.
type Client struct {
	api   anthropic.Client
	model string
	http  *http.Client
}

// NewClient builds the client. It makes no call until the first read.
func NewClient(cfg Config) *Client {
	c := &Client{
		model: cfg.Model,
		http:  &http.Client{Timeout: readTimeout},
	}
	if c.model == "" {
		c.model = DefaultModel
	}
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithHTTPClient(c.http),
		option.WithMaxRetries(0),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	c.api = anthropic.NewClient(opts...)
	return c
}

// wireFridge is the JSON the model answers a fridge read in (fridgeSchema). The pointers tell a
// missing field from an empty one, so an answer without its list or its roast is refused.
type wireFridge struct {
	Items *[]wireItem `json:"items"`
	Roast *string     `json:"roast"`
}

type wireItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity string `json:"quantity"`
}

// ReadFridge sends the photos (JPEGs, as the API accepts them) and the roast level to the model,
// and returns what it listed and its roast. Any failure of the model, including an answer that
// does not decode, is ErrUnavailable, except a caller that gave up, which gets its own ctx error.
func (c *Client) ReadFridge(ctx context.Context, photos [][]byte, level RoastLevel) (Fridge, error) {
	const op = "vision: read fridge"

	// 1. The roast level's line. An unknown level is our bug, not the model's
	roast, err := roastPrompt(level)
	if err != nil {
		return Fridge{}, err
	}

	// 2. The photos, then the level, in one user turn
	content := make([]anthropic.BetaContentBlockParamUnion, 0, len(photos)+1)
	for _, photo := range photos {
		content = append(content, anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{
			Data:      base64.StdEncoding.EncodeToString(photo),
			MediaType: anthropic.BetaBase64ImageSourceMediaTypeImageJPEG,
		}))
	}
	content = append(content, anthropic.NewBetaTextBlock(roast))

	// 3. One call. Low effort keeps the judging screen quick, and the default fallback lets
	// another model answer if this one declines
	msg, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     c.model,
		MaxTokens: readMaxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: fridgePrompt}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(content...)},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortLow,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: fridgeSchema},
		},
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return Fridge{}, c.fail(ctx, op, err)
	}

	// 4. An answer that stopped early or was declined has no fridge in it
	switch msg.StopReason {
	case anthropic.BetaStopReasonEndTurn:
	case anthropic.BetaStopReasonRefusal:
		return Fridge{}, fmt.Errorf("%s: model declined: %w", op, ErrUnavailable)
	default:
		return Fridge{}, fmt.Errorf("%s: stopped on %q: %w", op, msg.StopReason, ErrUnavailable)
	}

	// 5. Decode the whole answer or none of it
	var text strings.Builder
	for _, block := range msg.Content {
		b, ok := block.AsAny().(anthropic.BetaTextBlock)
		if ok {
			text.WriteString(b.Text)
		}
	}
	fridge, err := decodeFridge(text.String())
	if err != nil {
		return Fridge{}, fmt.Errorf("%s: %w: %w", op, err, ErrUnavailable)
	}
	return fridge, nil
}

// decodeFridge reads the model's JSON into a Fridge, refusing unknown fields, trailing data and a
// missing list or roast.
func decodeFridge(text string) (Fridge, error) {
	dec := json.NewDecoder(bytes.NewReader([]byte(text)))
	dec.DisallowUnknownFields()
	var w wireFridge
	err := dec.Decode(&w)
	if err != nil {
		return Fridge{}, fmt.Errorf("decode answer: %w", err)
	}
	if dec.More() {
		return Fridge{}, errors.New("decode answer: data after the object")
	}
	switch {
	case w.Items == nil:
		return Fridge{}, errors.New("decode answer: no items")
	case w.Roast == nil || strings.TrimSpace(*w.Roast) == "":
		return Fridge{}, errors.New("decode answer: no roast")
	}
	items := make([]Item, 0, len(*w.Items))
	for _, it := range *w.Items {
		items = append(items, Item(it))
	}
	return Fridge{Items: items, Roast: *w.Roast}, nil
}

// fail turns a failed call into what the caller branches on. A caller that gave up gets its ctx
// error. Anything else, our own timeout included, is ErrUnavailable, and the cause stays text so
// a client timeout is not mistaken for the caller's deadline.
func (c *Client) fail(ctx context.Context, op string, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w", op, ctx.Err())
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: status %d: %w", op, apiErr.StatusCode, ErrUnavailable)
	}
	//nolint:errorlint // a client timeout matches context.DeadlineExceeded, which ConnectErr would answer before ErrUnavailable
	return fmt.Errorf("%s: %v: %w", op, err, ErrUnavailable)
}
