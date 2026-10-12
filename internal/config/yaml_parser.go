package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// YAMLParser parses YAML configuration files while preserving mapping order.
type YAMLParser struct{}

// Parse decodes YAML into normalized raw values.
func (YAMLParser) Parse(path string, data []byte) (any, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	// Validate aliases, duplicate keys and merge operands with the YAML decoder
	// before traversing nodes to preserve source order.
	var decoded any
	if err := node.Decode(&decoded); err != nil {
		return nil, err
	}
	return yamlNodeValue(&node), nil
}

func yamlNodeValue(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil
		}
		return yamlNodeValue(node.Content[0])
	case yaml.SequenceNode:
		out := make([]any, 0, len(node.Content))
		for _, item := range node.Content {
			out = append(out, yamlNodeValue(item))
		}
		return out
	case yaml.MappingNode:
		return yamlMappingValue(node)
	case yaml.ScalarNode:
		var value any
		if err := node.Decode(&value); err == nil {
			return value
		}
		return node.Value
	case yaml.AliasNode:
		return yamlNodeValue(node.Alias)
	default:
		return nil
	}
}

func yamlMappingValue(node *yaml.Node) orderedMap {
	explicit := map[string]bool{}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Tag != "!!merge" {
			explicit[fmt.Sprint(yamlNodeValue(node.Content[i]))] = true
		}
	}
	out := orderedMap{}
	merged := map[string]bool{}
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Tag != "!!merge" {
			out = append(out, orderedPair{key: fmt.Sprint(yamlNodeValue(key)), value: yamlNodeValue(value)})
			continue
		}
		sources := []any{yamlNodeValue(value)}
		if sequence, ok := sources[0].([]any); ok {
			sources = sequence
		}
		for _, source := range sources {
			pairs, _ := source.(orderedMap)
			for _, pair := range pairs {
				// Explicit keys win; earlier mappings in a merge sequence win.
				if !explicit[pair.key] && !merged[pair.key] {
					out = append(out, pair)
					merged[pair.key] = true
				}
			}
		}
	}
	return out
}

var _ Parser = YAMLParser{}
