package reports

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCSVAmount(t *testing.T) {
	require.Equal(t, "12,34", csvAmount(1234))
	require.Equal(t, "-12,05", csvAmount(-1205))
	require.Equal(t, "0,00", csvAmount(0))
}
