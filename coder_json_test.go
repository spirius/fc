package fc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var testInput = `{"asd":123,"asdf":[1,2,3],"bsd":"asd","complex":{"asd":123}}`

func TestJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, DefaultRecoder.Run(&Config{
		Decoder: "j",
		Encoder: "j",
		Input:   bytes.NewBufferString(testInput),
		Output:  &out,
	}))
	require.JSONEq(t, testInput, out.String())
}
