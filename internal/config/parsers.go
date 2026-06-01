package config

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gurkankaymak/hocon"
	"github.com/pelletier/go-toml/v2"
	"github.com/titanous/json5"
	"gopkg.in/yaml.v3"
)

type YAMLParser struct{}

func (YAMLParser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type JSONParser struct{}

func (JSONParser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type JSON5Parser struct{}

func (JSON5Parser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := json5.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type TOMLParser struct{}

func (TOMLParser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

type HOCONParser struct{}

func (HOCONParser) Parse(path string, data []byte) (any, error) {
	cfg, err := hocon.ParseString(string(data))
	if err != nil {
		return nil, err
	}
	return hoconValue(cfg.GetRoot()), nil
}

func hoconValue(value hocon.Value) any {
	switch v := value.(type) {
	case nil:
		return nil
	case hocon.Object:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = hoconValue(item)
		}
		return out
	case hocon.Array:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, hoconValue(item))
		}
		return out
	case hocon.String:
		return strings.Trim(string(v), `"`)
	case hocon.Int:
		return int(v)
	case hocon.Float32:
		return float32(v)
	case hocon.Float64:
		return float64(v)
	case hocon.Boolean:
		return bool(v)
	case hocon.Duration:
		return time.Duration(v)
	case hocon.Null:
		return nil
	default:
		return hoconScalar(value)
	}
}

func hoconScalar(value hocon.Value) any {
	switch value.Type() {
	case hocon.StringType:
		return strings.Trim(value.String(), `"`)
	case hocon.NumberType:
		if i, err := strconv.Atoi(value.String()); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return f
		}
	case hocon.BooleanType:
		if b, err := strconv.ParseBool(value.String()); err == nil {
			return b
		}
	case hocon.NullType:
		return nil
	}
	return value.String()
}

var (
	_ Parser = YAMLParser{}
	_ Parser = JSONParser{}
	_ Parser = JSON5Parser{}
	_ Parser = TOMLParser{}
	_ Parser = HOCONParser{}
)
