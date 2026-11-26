package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	require.Equal(t, zapcore.InfoLevel, logLevel(), "tests log without the SQL debug output")

	t.Setenv("LOG_LEVEL", "debug")
	require.Equal(t, zapcore.DebugLevel, logLevel())

	t.Setenv("LOG_LEVEL", "warn")
	require.Equal(t, zapcore.WarnLevel, logLevel())

	t.Setenv("LOG_LEVEL", "nonsense")
	require.Equal(t, zapcore.InfoLevel, logLevel())
}
