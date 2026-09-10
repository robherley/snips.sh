package testutil

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

type LogRecorder struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *LogRecorder) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (l *LogRecorder) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, r)
	return nil
}

func (l *LogRecorder) WithAttrs(_ []slog.Attr) slog.Handler { return l }
func (l *LogRecorder) WithGroup(_ string) slog.Handler      { return l }

func (l *LogRecorder) AssertLog(t *testing.T, lvl slog.Level, substr string) {
	if !l.hasLog(t, lvl, substr) {
		t.Errorf("expected log with level %v containing %q, but none found", lvl, substr)
	}
}

func (l *LogRecorder) RefuteLog(t *testing.T, lvl slog.Level, substr string) {
	if l.hasLog(t, lvl, substr) {
		t.Errorf("unexpected log with level %v containing %q", lvl, substr)
	}
}

func (l *LogRecorder) hasLog(t *testing.T, lvl slog.Level, substr string) bool {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.records {
		if r.Level == lvl && strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

func SetLogRecorder(t *testing.T) *LogRecorder {
	t.Helper()
	h := &LogRecorder{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}
