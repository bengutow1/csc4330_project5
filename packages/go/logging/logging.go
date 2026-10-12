// Author: Caden Lund
// Package logging builds the structured logger every binary shares, and the HTTP middleware that
// logs through it.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New returns the logger for one binary: JSON on stdout (text with LOG_FORMAT=text), INFO and
// above (debug too with LOG_LEVEL=debug), and every line stamped with the service and
// APP_VERSION. main builds it once, sets it as the slog default and injects it everywhere else.
func New(service string) *slog.Logger {
	return build(os.Stdout, settings{
		service: service,
		format:  os.Getenv("LOG_FORMAT"),
		level:   os.Getenv("LOG_LEVEL"),
		version: os.Getenv("APP_VERSION"),
	})
}

// settings is what New reads from the environment, kept apart so a test can build a logger
// without touching process state.
type settings struct {
	service string
	format  string // "text", else JSON
	level   string // "debug", else INFO
	version string // the git sha baked into the image; "" is a local build
}

func build(w io.Writer, s settings) *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(s.level, "debug") {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if strings.EqualFold(s.format, "text") {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}

	version := s.version
	if version == "" {
		version = "dev"
	}
	return slog.New(h).With("service", s.service, "version", version)
}
