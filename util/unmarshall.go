package util

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func UnmarshallJsonBody[U any](t *testing.T, responseBody *bytes.Buffer) U {
	data, err := io.ReadAll(responseBody)
	require.NoError(t, err)

	var responseAccount U

	err = json.Unmarshal(data, &responseAccount)
	require.NoError(t, err)

	return responseAccount
}
