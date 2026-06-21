package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type YAMLParser struct{}

func (YAMLParser) Parse(path string, data []byte) (any, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
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
		out := make(orderedMap, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			out = append(out, orderedPair{
				key:   fmt.Sprint(yamlNodeValue(node.Content[i])),
				value: yamlNodeValue(node.Content[i+1]),
			})
		}
		return out
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

var _ Parser = YAMLParser{}
