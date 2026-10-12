// Author: Caden Lund
package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestEveryLineCarriesTheServiceAndVersion guards the two fields every log search filters on.
func TestEveryLineCarriesTheServiceAndVersion(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := build(&buf, settings{service: "api", version: "abc1234"})

	log.Info("scan judged", "scan_id", 7)

	var line map[string]any
	err := json.Unmarshal(buf.Bytes(), &line)
	if err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	if line["service"] != "api" {
		t.Errorf("service = %v, want api", line["service"])
	}
	if line["version"] != "abc1234" {
		t.Errorf("version = %v, want abc1234", line["version"])
	}
	if line["msg"] != "scan judged" {
		t.Errorf("msg = %v, want scan judged", line["msg"])
	}
	if line["scan_id"] != float64(7) {
		t.Errorf("scan_id = %v, want 7", line["scan_id"])
	}
}

// TestALocalBuildLogsVersionDev keeps a line from a laptop from passing for a release.
func TestALocalBuildLogsVersionDev(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := build(&buf, settings{service: "api"})

	log.Info("hello")

	if !strings.Contains(buf.String(), `"version":"dev"`) {
		t.Errorf("line = %s, want version dev", buf.String())
	}
}

// TestDebugLinesNeedLogLevelDebug keeps debug noise out of the log unless someone asked for it.
func TestDebugLinesNeedLogLevelDebug(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		level string
		want  bool
	}{
		{name: "unset", level: "", want: false},
		{name: "debug", level: "debug", want: true},
		{name: "any case", level: "DEBUG", want: true},
		{name: "anything else", level: "verbose", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := build(&buf, settings{service: "api", level: c.level})

			log.Debug("scoring detail")

			got := buf.Len() > 0
			if got != c.want {
				t.Errorf("LOG_LEVEL=%q printed debug = %v, want %v", c.level, got, c.want)
			}
		})
	}
}

// TestTextFormatIsOptIn keeps JSON, which the log pipeline parses, as the default.
func TestTextFormatIsOptIn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		format string
		prefix string
	}{
		{name: "unset is json", format: "", prefix: "{"},
		{name: "text", format: "text", prefix: "time="},
		{name: "any case", format: "TEXT", prefix: "time="},
		{name: "anything else is json", format: "yaml", prefix: "{"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := build(&buf, settings{service: "api", format: c.format})

			log.Info("hello")

			if !strings.HasPrefix(buf.String(), c.prefix) {
				t.Errorf("LOG_FORMAT=%q wrote %q, want it to start with %q", c.format, buf.String(), c.prefix)
			}
		})
	}
}
