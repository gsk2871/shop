package configreload

import (
	"context"
	"log/slog"
	"os"
)

// ConfigureLogging reads the currently published level and app name for every
// record. Request logs can use their pinned context to retain generation identity.
func (m *Manager) ConfigureLogging() {
	slog.SetDefault(slog.New(&runtimeLog{manager: m, inner: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})}))
}

type runtimeLog struct {
	manager *Manager
	inner   slog.Handler
}

func (h *runtimeLog) Enabled(ctx context.Context, level slog.Level) bool {
	c := h.manager.Config()
	if s, ok := FromContext(ctx); ok {
		c = s.Config
	}
	configured := slog.LevelInfo
	switch c.Log.Level {
	case "debug":
		configured = slog.LevelDebug
	case "warn":
		configured = slog.LevelWarn
	case "error":
		configured = slog.LevelError
	}
	return level >= configured
}
func (h *runtimeLog) Handle(ctx context.Context, r slog.Record) error {
	c := h.manager.Config()
	if s, ok := FromContext(ctx); ok {
		c = s.Config
	}
	r.AddAttrs(slog.String("app", c.App.Name))
	return h.inner.Handle(ctx, r)
}
func (h *runtimeLog) WithAttrs(a []slog.Attr) slog.Handler {
	return &runtimeLog{h.manager, h.inner.WithAttrs(a)}
}
func (h *runtimeLog) WithGroup(g string) slog.Handler {
	return &runtimeLog{h.manager, h.inner.WithGroup(g)}
}
