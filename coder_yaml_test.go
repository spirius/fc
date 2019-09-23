package fc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYAML(t *testing.T) {
	var out1 bytes.Buffer
	var out2 bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "j",
		Encoder: "y",
		Input:   bytes.NewBufferString(testInput),
		Output:  &out1,
	}))
	require.Equal(t, out1.String(), `asd: 123
asdf:
- 1
- 2
- 3
bsd: asd
complex:
  asd: 123
`)
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "y",
		Encoder: "j",
		Input:   &out1,
		Output:  &out2,
	}))
	require.JSONEq(t, testInput, out2.String())
}
