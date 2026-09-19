package utils

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestAlertCoreReportsFatalAndPanic guards against the bug this core replaces:
// the old notification pipeline matched the literal substring "ERROR" in the
// rendered log line, so FATAL/PANIC/DPANIC entries (which never contain that
// substring) were silently dropped. alertCore must not depend on level text at
// all, so it has to treat every level at/above DPanic as alert-worthy.
func TestAlertCoreReportsFatalAndPanic(t *testing.T) {
	core := newAlertCore(zap.NewAtomicLevelAt(zap.ErrorLevel))

	for _, level := range []zapcore.Level{zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel} {
		if !core.Enabled(level) {
			t.Errorf("level %s: expected alertCore to be enabled", level)
		}
	}
	if core.Enabled(zapcore.WarnLevel) {
		t.Error("expected alertCore to stay disabled below Error level")
	}
}

// TestAlertCoreWriteDoesNotPanicWithoutSentryOrEmailConfigured exercises the
// actual Write path (including the Fatal-only synchronous notification branch)
// with neither Sentry nor email configured - the common case for a local run -
// and asserts it degrades to a no-op rather than panicking or blocking.
func TestAlertCoreWriteDoesNotPanicWithoutSentryOrEmailConfigured(t *testing.T) {
	core := newAlertCore(zap.NewAtomicLevelAt(zap.ErrorLevel))

	entry := zapcore.Entry{
		Level:   zapcore.FatalLevel,
		Time:    time.Now(),
		Message: "db init failed",
		Caller:  zapcore.NewEntryCaller(0, "utils/alert_core_test.go", 42, true),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := core.Write(entry, []zapcore.Field{zap.Int("order_id", 9)}); err != nil {
			t.Errorf("Write returned error: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("alertCore.Write did not return - Fatal-path notification must stay synchronous but bounded")
	}
}
