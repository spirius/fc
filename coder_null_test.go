package fc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNULL(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "n",
		Encoder: "j",
		Input:   nil,
		Output:  &out,
	}))
	require.Equal(t, "null\n", out.String())
}
