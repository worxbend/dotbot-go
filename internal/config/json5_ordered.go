package config

import (
	"fmt"

	"github.com/titanous/json5"
)

// JSON5Parser parses JSON5 configuration files while preserving object order.
type JSON5Parser struct{}

// Parse decodes JSON5 into normalized raw values.
func (JSON5Parser) Parse(path string, data []byte) (any, error) {
	var raw any
	if err := json5.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	ordered, err := json5OrderedValue(data)
	if err != nil {
		return nil, err
	}
	return ordered, nil
}

func json5OrderedValue(data []byte) (any, error) {
	start, err := skipJSON5Space(data, 0)
	if err != nil {
		return nil, err
	}
	if start >= len(data) {
		var raw any
		if err := json5.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		return raw, nil
	}

	switch data[start] {
	case '[':
		return json5OrderedArray(data[start:])
	case '{':
		return json5OrderedObject(data[start:])
	default:
		var raw any
		if err := json5.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		return raw, nil
	}
}

func json5OrderedArray(data []byte) ([]any, error) {
	values := []any{}
	i := 1

	for {
		var err error
		i, err = skipJSON5Space(data, i)
		if err != nil {
			return nil, err
		}
		if i >= len(data) {
			return nil, fmt.Errorf("unterminated json5 array")
		}
		if data[i] == ']' {
			return values, nil
		}

		end, closing, err := json5ValueEnd(data, i, ']')
		if err != nil {
			return nil, err
		}
		value, err := json5OrderedValue(data[i:end])
		if err != nil {
			return nil, err
		}
		values = append(values, value)

		if closing {
			return values, nil
		}
		i = end + 1
	}
}

func json5OrderedObject(data []byte) (orderedMap, error) {
	pairs := orderedMap{}
	i := 1

	for {
		var err error
		i, err = skipJSON5Space(data, i)
		if err != nil {
			return nil, err
		}
		if i >= len(data) {
			return nil, fmt.Errorf("unterminated json5 object")
		}
		if data[i] == '}' {
			return pairs, nil
		}

		colon, err := json5ObjectKeyEnd(data, i)
		if err != nil {
			return nil, err
		}
		key, err := json5ObjectKey(data[i:colon])
		if err != nil {
			return nil, err
		}

		valueStart, err := skipJSON5Space(data, colon+1)
		if err != nil {
			return nil, err
		}
		if valueStart >= len(data) {
			return nil, fmt.Errorf("json5 object key %q is missing a value", key)
		}

		end, closing, err := json5ValueEnd(data, valueStart, '}')
		if err != nil {
			return nil, err
		}
		value, err := json5OrderedValue(data[valueStart:end])
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, orderedPair{
			key:   key,
			value: value,
		})

		if closing {
			return pairs, nil
		}
		i = end + 1
	}
}

func json5ObjectKey(data []byte) (string, error) {
	doc := make([]byte, 0, len(data)+8)
	doc = append(doc, '{')
	doc = append(doc, data...)
	doc = append(doc, ':', 'n', 'u', 'l', 'l', '}')

	var object map[string]any
	if err := json5.Unmarshal(doc, &object); err != nil {
		return "", err
	}
	for key := range object {
		return key, nil
	}
	return "", fmt.Errorf("json5 object key is empty")
}

type json5ScanState struct {
	quote        byte
	escaped      bool
	lineComment  bool
	blockComment bool
}

// consumeHidden advances over bytes that should be ignored by delimiter
// scanning because they are inside a JSON5 string or comment.
func (s *json5ScanState) consumeHidden(data []byte, i int) (int, bool) {
	c := data[i]

	switch {
	case s.lineComment:
		if c == '\n' || c == '\r' {
			s.lineComment = false
		}
		return i, true
	case s.blockComment:
		if c == '*' && i+1 < len(data) && data[i+1] == '/' {
			s.blockComment = false
			return i + 1, true
		}
		return i, true
	case s.quote != 0:
		if s.escaped {
			s.escaped = false
			return i, true
		}
		if c == '\\' {
			s.escaped = true
			return i, true
		}
		if c == s.quote {
			s.quote = 0
		}
		return i, true
	}

	if c == '"' || c == '\'' {
		s.quote = c
		return i, true
	}
	if c == '/' && i+1 < len(data) {
		switch data[i+1] {
		case '/':
			s.lineComment = true
			return i + 1, true
		case '*':
			s.blockComment = true
			return i + 1, true
		}
	}
	return i, false
}

func json5ObjectKeyEnd(data []byte, start int) (int, error) {
	state := json5ScanState{}

	for i := start; i < len(data); i++ {
		c := data[i]
		if next, ok := state.consumeHidden(data, i); ok {
			i = next
			continue
		}
		if c == ':' {
			return i, nil
		}
		if c == '}' || c == ',' {
			return 0, fmt.Errorf("json5 object key is missing a colon")
		}
	}

	return 0, fmt.Errorf("unterminated json5 object key")
}

func json5ValueEnd(data []byte, start int, closing byte) (end int, closed bool, err error) {
	depth := 0
	state := json5ScanState{}

	for i := start; i < len(data); i++ {
		c := data[i]
		if next, ok := state.consumeHidden(data, i); ok {
			i = next
			continue
		}

		switch c {
		case '[', '{':
			depth++
		case ']':
			if depth == 0 && closing == ']' {
				return i, true, nil
			}
			depth--
		case '}':
			if depth == 0 && closing == '}' {
				return i, true, nil
			}
			depth--
		case ',':
			if depth == 0 {
				return i, false, nil
			}
		}
		if depth < 0 {
			return 0, false, fmt.Errorf("unexpected json5 closing delimiter %q", c)
		}
	}

	return 0, false, fmt.Errorf("unterminated json5 value")
}

func skipJSON5Space(data []byte, start int) (int, error) {
	for i := start; i < len(data); i++ {
		c := data[i]
		switch {
		case isJSON5Space(c):
			continue
		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			i += 2
			for i < len(data) && data[i] != '\n' && data[i] != '\r' {
				i++
			}
			if i < len(data) {
				i--
			}
		case c == '/' && i+1 < len(data) && data[i+1] == '*':
			i += 2
			for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			if i+1 >= len(data) {
				return 0, fmt.Errorf("unterminated json5 block comment")
			}
			i++
		default:
			return i, nil
		}
	}
	return len(data), nil
}

func isJSON5Space(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\f':
		return true
	default:
		return false
	}
}

var _ Parser = JSON5Parser{}
