package common

import (
	"bytes"
	"encoding/json"
	"io"

	gojson "github.com/goccy/go-json"
)

func Unmarshal(data []byte, v any) error {
	return gojson.Unmarshal(data, v)
}

func UnmarshalJsonStr(data string, v any) error {
	return gojson.Unmarshal([]byte(data), v)
}

func DecodeJson(reader io.Reader, v any) error {
	return gojson.NewDecoder(reader).Decode(v)
}

func Marshal(v any) ([]byte, error) {
	return gojson.Marshal(v)
}

func GetJsonType(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}
