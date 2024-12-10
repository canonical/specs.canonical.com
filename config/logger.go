package config

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/fatih/color"
)

func SetupLogger() *slog.Logger {
	appEnv := GetEnv("APP_ENV")
	isProduction := strings.ToLower(appEnv) == "production"

	var handler slog.Handler
	if isProduction {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = NewPrettyHandler(os.Stdout, PrettyHandlerOptions{
			SlogOpts: slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

type PrettyHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type PrettyHandler struct {
	opts   PrettyHandlerOptions
	l      *log.Logger
	attrs  []slog.Attr
	groups []string
}

// Ensure PrettyHandler implements slog.Handler
var _ slog.Handler = (*PrettyHandler)(nil)

func NewPrettyHandler(out io.Writer, opts PrettyHandlerOptions) *PrettyHandler {
	return &PrettyHandler{
		opts:  opts,
		l:     log.New(out, "", 0),
		attrs: make([]slog.Attr, 0),
	}
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	// Combine pre-recorded attrs with record attrs
	fields := make(map[string]any)

	// Add pre-recorded attrs
	for _, attr := range h.attrs {
		fields[attr.Key] = attr.Value.Any()
	}

	// Add record attrs
	r.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = attr.Value.Any()
		return true
	})

	var formattedFields string
	if len(fields) > 0 {
		b, err := json.MarshalIndent(fields, "", " ")
		if err != nil {
			return err
		}
		formattedFields = string(b)
	}

	timeStr := color.GreenString(r.Time.Format("[15:04:05]"))
	msg := color.WhiteString(r.Message)

	if formattedFields != "" {
		formattedFields = color.CyanString(formattedFields)
	}

	h.l.Println(level, timeStr, msg, formattedFields)
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{
		opts:   h.opts,
		l:      h.l,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{
		opts:   h.opts,
		l:      h.l,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

// Enabled implements Handler.Enabled
func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	configuredLogLevel := GetEnv("LOG_LEVEL")
	configuredLogLevel = strings.ToLower(configuredLogLevel)

	switch configuredLogLevel {
	case "debug":
		return level >= slog.LevelDebug
	case "info":
		return level >= slog.LevelInfo
	case "warn":
		return level >= slog.LevelWarn
	case "error":
		return level >= slog.LevelError
	default:
		return level >= slog.LevelDebug
	}
}
