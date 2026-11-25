package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	require.Error(t, config{}.Validate())
	require.Error(t, config{Port: "3000"}.Validate())
	require.NoError(t, config{Port: "3000", FrontendURL: "https://shop.example.com"}.Validate())
}

func TestEnvHelpersFallBackOnInvalidValues(t *testing.T) {
	t.Setenv("TEST_FLOAT", "not a number")
	t.Setenv("TEST_INT", "many")
	require.Equal(t, 1.5, getEnvFloat("TEST_FLOAT", 1.5))
	require.Equal(t, 7, getEnvInt("TEST_INT", 7))

	t.Setenv("TEST_FLOAT", "2.25")
	t.Setenv("TEST_INT", "9")
	require.Equal(t, 2.25, getEnvFloat("TEST_FLOAT", 1.5))
	require.Equal(t, 9, getEnvInt("TEST_INT", 7))
	require.Equal(t, 3.5, getEnvFloat("TEST_FLOAT_UNSET", 3.5))

	t.Setenv("TEST_LIST", " a, ,b ")
	require.Equal(t, []string{"a", "b"}, getEnvStringSlice("TEST_LIST", ""))
}
