package fc

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTPL(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder:     "j",
		Encoder:     "tpl",
		EncoderArgs: []string{"./testdata/main.tpl"},
		Input:       bytes.NewBufferString(`{"list": ["a", "b", "c"], "map": {"k1": "v1", "k2": "v2"}}`),
		Output:      &out,
	}))
	require.Equal(t, `0:a
1:b
2:c
k1:v1
k2:v2
test
test`, out.String())
}

func TestTPLInputJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder:     "j",
		Encoder:     "tpl",
		EncoderArgs: []string{"./testdata/input.tpl"},
		Input:       bytes.NewBufferString(testInput),
		Output:      &out,
	}))
	require.JSONEq(t, testInput, out.String())
}

func TestTPLOutputJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder:     "j",
		Encoder:     "tpl",
		EncoderArgs: []string{"./testdata/output.tpl"},
		Input:       bytes.NewBufferString(testInput),
		Output:      &out,
	}))
	require.JSONEq(t, testInput, out.String())
}

func TestTPLImport(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder:     "j",
		Encoder:     "tpl",
		EncoderArgs: []string{"./testdata/import.tpl"},
		Input:       bytes.NewBufferString(testInput),
		Output:      &out,
	}))
	expOutput, err := ioutil.ReadFile("./testdata/import/basic/file1.json")
	require.NoError(t, err)
	require.JSONEq(t, string(expOutput), out.String())
}
