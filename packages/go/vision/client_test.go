// Author: Ben Gutowski
package vision_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bengutow1/csc4330_project5/packages/go/vision"
)

const testKey = "test_key"

// vendor is a stand-in for Anthropic's API that answers every call with one recorded response,
// and records what it was sent. Safe as a fake: it is the true edge (go-vendors.md), and it
// replays the vendor's own wire format.
type vendor struct {
	mu     sync.Mutex
	status int
	body   []byte
	delay  time.Duration
	calls  []call
}

// call is one request the vendor received.
type call struct {
	path   string
	key    string
	betas  string
	body   map[string]any
	rawLen int
}

func (v *vendor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body map[string]any
	_ = json.Unmarshal(raw, &body) // a body that does not decode is recorded as nil, and asserted on
	v.mu.Lock()
	v.calls = append(v.calls, call{
		path: r.URL.Path, key: r.Header.Get("X-Api-Key"), betas: r.Header.Get("Anthropic-Beta"),
		body: body, rawLen: len(raw),
	})
	status, resp, delay := v.status, v.body, v.delay
	v.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(resp) // the client under test reports what it got
}

func (v *vendor) recorded() []call {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.calls)
}

// newVendor serves the recorded response in testdata/name with status, and a client pointed at it.
func newVendor(t *testing.T, status int, name string) (*vendor, *vision.Client) {
	t.Helper()
	v := recorded(t, name)
	v.status = status
	return serve(t, v)
}

func serve(t *testing.T, v *vendor) (*vendor, *vision.Client) {
	t.Helper()
	srv := httptest.NewServer(v)
	t.Cleanup(srv.Close)
	return v, vision.NewClient(vision.Config{APIKey: testKey, Model: "test-model", BaseURL: srv.URL})
}

// photos stands in for a scan's JPEGs. The client never looks inside them.
var photos = [][]byte{[]byte("fridge photo bytes"), []byte("pantry photo bytes")}

// TestAGoodAnswerBecomesTheFridgeTheModelListed: the verdict must carry exactly what the model
// listed and its roast, so the scorer judges the real list.
func TestAGoodAnswerBecomesTheFridgeTheModelListed(t *testing.T) {
	t.Parallel()
	_, client := newVendor(t, http.StatusOK, "good_fridge.json")

	fridge, err := client.ReadFridge(t.Context(), photos, vision.RoastMedium)
	if err != nil {
		t.Fatalf("read fridge: %v", err)
	}
	want := []vision.Item{
		{Name: "Eggs", Category: "protein", Quantity: "x6"},
		{Name: "Spinach", Category: "vegetable", Quantity: "1 bag"},
		{Name: "Hot sauce", Category: "condiment", Quantity: ""},
	}
	if !slices.Equal(fridge.Items, want) {
		t.Errorf("items = %+v, want %+v", fridge.Items, want)
	}
	wantRoast := "Eggs, spinach and hot sauce. This fridge has a plan, it just has not told anyone yet."
	if fridge.Roast != wantRoast {
		t.Errorf("roast = %q, want %q", fridge.Roast, wantRoast)
	}
}

// TestAnEmptyFridgeIsAVerdictWithNoItems: no food is still an answer (it scores 0), not a failure.
func TestAnEmptyFridgeIsAVerdictWithNoItems(t *testing.T) {
	t.Parallel()
	_, client := newVendor(t, http.StatusOK, "empty_fridge.json")

	fridge, err := client.ReadFridge(t.Context(), photos, vision.RoastExtraCrispy)
	if err != nil {
		t.Fatalf("read fridge: %v", err)
	}
	if len(fridge.Items) != 0 {
		t.Errorf("items = %+v, want none", fridge.Items)
	}
	if fridge.Roast != "Just shelves and cold air. This fridge is mostly a light bulb." {
		t.Errorf("roast = %q, want the empty fridge's roast", fridge.Roast)
	}
}

// TestTheRequestCarriesThePhotosTheLevelAndTheShapeAndNothingElse: the prompt must carry the roast
// level and ask for the strict shape, the photos must all arrive as JPEGs, and nothing about the
// player may go to the vendor.
func TestTheRequestCarriesThePhotosTheLevelAndTheShapeAndNothingElse(t *testing.T) {
	t.Parallel()
	v, client := newVendor(t, http.StatusOK, "good_fridge.json")

	_, err := client.ReadFridge(t.Context(), photos, vision.RoastMild)
	if err != nil {
		t.Fatalf("read fridge: %v", err)
	}
	calls := v.recorded()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	c := calls[0]
	if c.path != "/v1/messages" {
		t.Errorf("path = %q, want /v1/messages", c.path)
	}
	if c.key != testKey {
		t.Errorf("x-api-key = %q, want the configured key", c.key)
	}
	if !strings.Contains(c.betas, "server-side-fallback-2026-07-01") {
		t.Errorf("anthropic-beta = %q, want the server-side fallback beta", c.betas)
	}
	if c.body == nil {
		t.Fatalf("request body did not decode (%d bytes)", c.rawLen)
	}
	if c.body["model"] != "test-model" {
		t.Errorf("model = %v, want test-model", c.body["model"])
	}
	if c.body["fallbacks"] != "default" {
		t.Errorf("fallbacks = %v, want default", c.body["fallbacks"])
	}

	// The strict shape, at low effort
	out, _ := c.body["output_config"].(map[string]any)
	format, _ := out["format"].(map[string]any)
	if format["type"] != "json_schema" || format["schema"] == nil {
		t.Errorf("output_config.format = %v, want a json_schema", format)
	}
	if out["effort"] != "low" {
		t.Errorf("effort = %v, want low", out["effort"])
	}

	// The system prompt keeps roasts off the person
	system, _ := json.Marshal(c.body["system"])
	if !strings.Contains(string(system), "never the person") {
		t.Errorf("system prompt = %s, want the rule that roasts are never about the person", system)
	}

	// One user turn: each photo as a JPEG, then the level
	msgs, _ := c.body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1", len(msgs))
	}
	content, _ := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != len(photos)+1 {
		t.Fatalf("content blocks = %d, want %d photos and the level", len(content), len(photos))
	}
	for i, photo := range photos {
		block, _ := content[i].(map[string]any)
		source, _ := block["source"].(map[string]any)
		if block["type"] != "image" || source["media_type"] != "image/jpeg" {
			t.Errorf("block %d = %v, want a JPEG image", i, block)
		}
		if source["data"] != base64.StdEncoding.EncodeToString(photo) {
			t.Errorf("block %d data is not photo %d", i, i)
		}
	}
	level, _ := content[len(photos)].(map[string]any)
	if text, _ := level["text"].(string); !strings.HasPrefix(text, "Roast level: mild.") {
		t.Errorf("last block = %v, want the mild roast level", level)
	}
}

// TestEveryRoastLevelIsSentAndAnUnknownOneIsRefused: the player's level must reach the model, and
// a level outside the three must never be quietly served as another.
func TestEveryRoastLevelIsSentAndAnUnknownOneIsRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		level vision.RoastLevel
		want  string
	}{
		{name: "mild", level: vision.RoastMild, want: "Roast level: mild."},
		{name: "medium", level: vision.RoastMedium, want: "Roast level: medium."},
		{name: "extra crispy", level: vision.RoastExtraCrispy, want: "Roast level: extra crispy."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v, client := newVendor(t, http.StatusOK, "good_fridge.json")
			_, err := client.ReadFridge(t.Context(), photos, c.level)
			if err != nil {
				t.Fatalf("read fridge: %v", err)
			}
			calls := v.recorded()
			if len(calls) != 1 {
				t.Fatalf("calls = %d, want 1", len(calls))
			}
			raw, _ := json.Marshal(calls[0].body["messages"])
			if !strings.Contains(string(raw), c.want) {
				t.Errorf("messages = %s, want %q", raw, c.want)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()
		v, client := newVendor(t, http.StatusOK, "good_fridge.json")
		_, err := client.ReadFridge(t.Context(), photos, vision.RoastLevel("spicy"))
		if err == nil {
			t.Fatal("read fridge at an unknown level succeeded, want an error")
		}
		if errors.Is(err, vision.ErrUnavailable) {
			t.Errorf("err = %v, want our own bug, not the model being unavailable", err)
		}
		if n := len(v.recorded()); n != 0 {
			t.Errorf("calls = %d, want none spent on a level we cannot send", n)
		}
	})
}

// TestAnAnswerThatDoesNotDecodeIsUnavailableNeverAPartialFridge: a malformed or partial answer
// must never become a verdict, and the player must be told to try again.
func TestAnAnswerThatDoesNotDecodeIsUnavailableNeverAPartialFridge(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		vendor *vendor
	}{
		{name: "body is not json", vendor: &vendor{status: http.StatusOK, body: []byte("<html>bad gateway</html>")}},
		{name: "answer has no roast", vendor: recorded(t, "answer_without_roast.json")},
		{name: "answer was cut off", vendor: recorded(t, "cut_off_answer.json")},
		{name: "model declined", vendor: recorded(t, "refusal.json")},
		{name: "answer has an unknown field", vendor: answering(t, `{"items":[],"roast":"Cold.","score":100}`)},
		{name: "answer has trailing data", vendor: answering(t, `{"items":[],"roast":"Cold."}{"items":[]}`)},
		{name: "answer is empty", vendor: answering(t, ``)},
		{name: "roast runs far past 40 words", vendor: answering(t, `{"items":[],"roast":"`+strings.Repeat("cold ", 61)+`"}`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, client := serve(t, c.vendor)
			fridge, err := client.ReadFridge(t.Context(), photos, vision.RoastMedium)
			if !errors.Is(err, vision.ErrUnavailable) {
				t.Errorf("err = %v, want ErrUnavailable", err)
			}
			if fridge.Items != nil || fridge.Roast != "" {
				t.Errorf("fridge = %+v, want none at all", fridge)
			}
		})
	}
}

// TestAProviderErrorIsUnavailableAndNeverRetried: a vendor outage must tell the player to try
// again, and the client must not multiply the bill by retrying on its own.
func TestAProviderErrorIsUnavailableAndNeverRetried(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
	}{
		{name: "overloaded", status: 529},
		{name: "server error", status: http.StatusInternalServerError},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "bad key", status: http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v, client := newVendor(t, c.status, "overloaded.json")
			fridge, err := client.ReadFridge(t.Context(), photos, vision.RoastMedium)
			if !errors.Is(err, vision.ErrUnavailable) {
				t.Errorf("err = %v, want ErrUnavailable", err)
			}
			if fridge.Items != nil || fridge.Roast != "" {
				t.Errorf("fridge = %+v, want none", fridge)
			}
			if n := len(v.recorded()); n != 1 {
				t.Errorf("calls = %d, want exactly 1 (no retries)", n)
			}
		})
	}
}

// TestAModelThatTakesTooLongIsUnavailableNotTheCallersDeadline: a hung vendor must end in "try
// again", and must not read as the caller's own deadline, which ConnectErr maps differently.
func TestAModelThatTakesTooLongIsUnavailableNotTheCallersDeadline(t *testing.T) {
	t.Parallel()
	slow := recorded(t, "good_fridge.json")
	slow.delay = 2 * time.Second
	_, client := serve(t, slow)
	client.WithTimeoutForTest(50 * time.Millisecond)

	_, err := client.ReadFridge(t.Context(), photos, vision.RoastMedium)
	if !errors.Is(err, vision.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want it not to match the caller's deadline", err)
	}
}

// TestACallerThatGivesUpGetsItsOwnErrorNotAnOutage: a player who hangs up is not a model outage,
// so it must not be logged or answered as one.
func TestACallerThatGivesUpGetsItsOwnErrorNotAnOutage(t *testing.T) {
	t.Parallel()
	slow := recorded(t, "good_fridge.json")
	slow.delay = 2 * time.Second
	_, client := serve(t, slow)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ReadFridge(ctx, photos, vision.RoastMedium)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want the caller's deadline", err)
	}
	if errors.Is(err, vision.ErrUnavailable) {
		t.Errorf("err = %v, want it not to read as a model outage", err)
	}
}

// recorded is a vendor that answers with testdata/name.
func recorded(t *testing.T, name string) *vendor {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read recorded response: %v", err)
	}
	return &vendor{status: http.StatusOK, body: body}
}

// answering is a vendor whose model answers with text, wrapped in the good response's envelope.
func answering(t *testing.T, text string) *vendor {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
		"content":     []any{map[string]any{"type": "text", "text": text}},
		"stop_reason": "end_turn", "stop_sequence": nil,
		"usage": map[string]any{"input_tokens": 1820, "output_tokens": 20},
	})
	if err != nil {
		t.Fatalf("build response: %v", err)
	}
	return &vendor{status: http.StatusOK, body: body}
}
