// Author: Caden Lund
package logging_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cadenlund/fries-in-the-bag/packages/go/logging"
)

// TestRequestLoggerRecordsTheStatusAndPathButNeverTheQuery: the query string is where a combo code
// or a token would ride, and those never reach a log.
func TestRequestLoggerRecordsTheStatusAndPathButNeverTheQuery(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logging.RequestLogger(jsonLog(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short and stout")) // a recorder never fails a write
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/combo/join?code=482913", nil))

	line := decode(t, &buf)
	if line["status"] != float64(http.StatusTeapot) {
		t.Errorf("status = %v, want %d", line["status"], http.StatusTeapot)
	}
	if line["path"] != "/combo/join" {
		t.Errorf("path = %v, want /combo/join", line["path"])
	}
	if line["bytes"] != float64(len("short and stout")) {
		t.Errorf("bytes = %v, want %d", line["bytes"], len("short and stout"))
	}
	if strings.Contains(buf.String(), "482913") {
		t.Errorf("the line carries the query string: %s", buf.String())
	}
}

// TestAHandlerThatNeverCallsWriteHeaderIsLoggedAs200 matches what net/http actually sent.
func TestAHandlerThatNeverCallsWriteHeaderIsLoggedAs200(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logging.RequestLogger(jsonLog(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok")) // a recorder never fails a write
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	line := decode(t, &buf)
	if line["status"] != float64(http.StatusOK) {
		t.Errorf("status = %v, want 200", line["status"])
	}
}

// TestRecovererAnswersAPanicWithA500AndLogsIt: without it the client sees a dropped connection and
// the stack lands outside the structured log.
func TestRecovererAnswersAPanicWithA500AndLogsIt(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logging.Recoverer(jsonLog(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("nil scan")
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/combo.v1.ComboService/JoinCombo", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("code = %d, want 500", rec.Code)
	}
	line := decode(t, &buf)
	if line["msg"] != "panic serving request" {
		t.Errorf("msg = %v, want panic serving request", line["msg"])
	}
	if line["panic"] != "nil scan" {
		t.Errorf("panic = %v, want nil scan", line["panic"])
	}
	if line["stack"] == "" || line["stack"] == nil {
		t.Error("the line has no stack")
	}
}

// TestRecovererLetsAnAbortedResponseKeepPanicking: http.ErrAbortHandler is a deliberate abandon,
// which net/http handles quietly, not a bug to log.
func TestRecovererLetsAnAbortedResponseKeepPanicking(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logging.Recoverer(jsonLog(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	got := serveRecovering(h)

	err, isErr := got.(error)
	if !isErr || !errors.Is(err, http.ErrAbortHandler) {
		t.Fatalf("recovered %v, want http.ErrAbortHandler to keep panicking", got)
	}
	if buf.Len() != 0 {
		t.Errorf("an aborted response was logged as a panic: %s", buf.String())
	}
}

// TestFlushReachesTheUnderlyingWriter: a server stream behind the logger must still deliver each
// message as it is sent.
func TestFlushReachesTheUnderlyingWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logging.RequestLogger(jsonLog(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("the logged writer is not an http.Flusher")
			return
		}
		f.Flush()
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !rec.Flushed {
		t.Error("Flush did not reach the underlying writer")
	}
}

func jsonLog(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, nil))
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var line map[string]any
	err := json.Unmarshal(buf.Bytes(), &line)
	if err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	return line
}

// serveRecovering serves one request and returns whatever panicked out of h, or nil.
func serveRecovering(h http.Handler) (recovered any) {
	defer func() { recovered = recover() }()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	return nil
}
