package fc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var testInputHCL = `
some_block "asd" {
  key = "value"
}

inputs = {
  str = "asd"
  map = {
    key1 = "value1"
  }
  ignored = func()
}

second = {
  list = [1,"a"]
}
`

func TestHCL(t *testing.T) {
	var out1 bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "h",
		Encoder: "j",
		Input:   bytes.NewBufferString(testInputHCL),
		Output:  &out1,
	}))
	require.JSONEq(t, `{"inputs":{"str": "asd", "map": {"key1":"value1"}, "ignored":null}, "second": {"list": [1, "a"]}}`, out1.String())
	var out2 bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "j",
		Encoder: "h",
		Input:   &out1,
		Output:  &out2,
	}))
	require.Contains(t, out2.String(), `inputs = { ignored = null, map = { key1 = "value1" }, str = "asd" }`)
	require.Contains(t, out2.String(), `second = { list = [1, "a"] }`)
}

func TestHCLMetadata(t *testing.T) {
	var out1 bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder:     "h",
		Encoder:     "tpl",
		EncoderArgs: []string{"./testdata/metadata.tpl"},
		Input:       bytes.NewBufferString(testInputHCL),
		Output:      &out1,
	}))
	require.JSONEq(t, `[{"attributes":{"key":"value"},"blocks":[],"labels":["asd"],"type":"some_block"}]`, out1.String())
}
