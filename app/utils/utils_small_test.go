package utils

import (
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestSentryLevel(t *testing.T) {
	require.Equal(t, sentry.LevelFatal, sentryLevel(zapcore.FatalLevel))
	require.Equal(t, sentry.LevelFatal, sentryLevel(zapcore.DPanicLevel))
	require.Equal(t, sentry.LevelError, sentryLevel(zapcore.ErrorLevel))
	require.Equal(t, sentry.LevelWarning, sentryLevel(zapcore.WarnLevel))
	require.Equal(t, sentry.LevelInfo, sentryLevel(zapcore.InfoLevel))
	require.Equal(t, sentry.LevelInfo, sentryLevel(zapcore.DebugLevel))
}

func TestHostOnly(t *testing.T) {
	require.Equal(t, "10.0.0.1", hostOnly("10.0.0.1:1234"))
	require.Equal(t, "::1", hostOnly("[::1]:80"))
	require.Equal(t, "10.0.0.1", hostOnly("10.0.0.1"))
}

func TestRandomIntAndUnixTime(t *testing.T) {
	require.Equal(t, 5, RandomInt(5, 5))
	require.Equal(t, 5, RandomInt(5, 1))
	for i := 0; i < 50; i++ {
		n := RandomInt(2, 4)
		require.GreaterOrEqual(t, n, 2)
		require.Less(t, n, 4)
	}
	require.InDelta(t, time.Now().Unix(), GetUnixTime(), 2)
}

func TestAlertCoreOnlyTakesEnabledLevels(t *testing.T) {
	core := newAlertCore(zap.NewAtomicLevelAt(zap.ErrorLevel))
	require.NoError(t, core.Sync())

	ce := core.Check(zapcore.Entry{Level: zapcore.InfoLevel}, nil)
	require.Nil(t, ce)
	ce = core.Check(zapcore.Entry{Level: zapcore.ErrorLevel}, nil)
	require.NotNil(t, ce)
}

func TestReqIDCoreSyncAndRequestID(t *testing.T) {
	obsCore, logs := observer.New(zap.DebugLevel)
	core := &reqIDCore{core: obsCore}
	require.NoError(t, core.Sync())

	SetRequestID("req-123")
	t.Cleanup(ClearRequestID)
	logger := zap.New(core)
	logger.Info("hello")

	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, "req-123", entries[0].ContextMap()["request_id"])
}
