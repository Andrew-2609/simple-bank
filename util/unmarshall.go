package util

import (
	"bytes"
	"encoding/json"
	"io"
)

func UnmarshallJsonBody[U any](responseBody *bytes.Buffer) (unmarshalledResponse U, err error) {
	data, err := io.ReadAll(responseBody)

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &unmarshalledResponse)

	return
}
