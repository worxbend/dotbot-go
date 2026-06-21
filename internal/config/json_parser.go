package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type JSONParser struct{}

func (JSONParser) Parse(path string, data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	raw, err := jsonValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected data after json document")
		}
		return nil, err
	}
	return raw, nil
}

func jsonValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	delim, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}

	switch delim {
	case '[':
		out := []any{}
		for decoder.More() {
			value, err := jsonValue(decoder)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		if err := expectJSONDelim(decoder, ']'); err != nil {
			return nil, err
		}
		return out, nil
	case '{':
		out := orderedMap{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("json object key must be a string")
			}
			value, err := jsonValue(decoder)
			if err != nil {
				return nil, err
			}
			out = append(out, orderedPair{key: key, value: value})
		}
		if err := expectJSONDelim(decoder, '}'); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unexpected json delimiter %q", delim)
	}
}

func expectJSONDelim(decoder *json.Decoder, expected json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != expected {
		return fmt.Errorf("expected json delimiter %q", expected)
	}
	return nil
}

var _ Parser = JSONParser{}
