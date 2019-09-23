package fc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTOML(t *testing.T) {
	var out1 bytes.Buffer
	var out2 bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "j",
		Encoder: "t",
		Input:   bytes.NewBufferString(testInput),
		Output:  &out1,
	}))
	require.Equal(t, out1.String(), `asd = 123.0
asdf = [1.0, 2.0, 3.0]
bsd = "asd"

[complex]
  asd = 123.0
`)
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "t",
		Encoder: "j",
		Input:   &out1,
		Output:  &out2,
	}))
	require.JSONEq(t, testInput, out2.String())
}
