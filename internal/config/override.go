package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// overrideSpec matches "dotted.path: <yaml value>". The value may contain colons
// (URLs) and spans the rest of the string, including newlines.
var overrideSpec = regexp.MustCompile(`^([A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)*)\s*:\s*(?s)(.*)$`)

// ApplyYAMLOverrides merges path assignments into raw gocloud YAML.
// Each override is "dotted.path: <yaml-value>". The value replaces that path.
// A YAML null (null or ~) removes the key. Later overrides win.
// Mapping key order of the original document is preserved.
func ApplyYAMLOverrides(data []byte, overrides []string) ([]byte, error) {
	if len(overrides) == 0 {
		return data, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("config YAML is empty")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config YAML root must be a mapping")
	}

	for _, spec := range overrides {
		path, value, err := parseOverride(spec)
		if err != nil {
			return nil, err
		}
		if err := setYAMLPath(root, path, value); err != nil {
			return nil, fmt.Errorf("--override %q: %w", strings.TrimSpace(spec), err)
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("failed to encode overridden config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode overridden config: %w", err)
	}
	return buf.Bytes(), nil
}

func parseOverride(spec string) ([]string, *yaml.Node, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, nil, fmt.Errorf("--override is empty; expected 'dotted.path: <yaml-value>'")
	}
	matches := overrideSpec.FindStringSubmatch(trimmed)
	if matches == nil {
		return nil, nil, fmt.Errorf("--override %q: expected 'dotted.path: <yaml-value>' (example: infrastructure.providers.use_profiles: false)", trimmed)
	}
	rawValue := matches[2]
	if strings.TrimSpace(rawValue) == "" {
		return nil, nil, fmt.Errorf("--override %q: missing YAML value", trimmed)
	}

	var valueDoc yaml.Node
	if err := yaml.Unmarshal([]byte(rawValue), &valueDoc); err != nil {
		return nil, nil, fmt.Errorf("--override %q: invalid YAML value: %w", trimmed, err)
	}
	if valueDoc.Kind != yaml.DocumentNode || len(valueDoc.Content) == 0 {
		return nil, nil, fmt.Errorf("--override %q: missing YAML value", trimmed)
	}

	return strings.Split(matches[1], "."), valueDoc.Content[0], nil
}

func setYAMLPath(root *yaml.Node, path []string, value *yaml.Node) error {
	if len(path) == 0 {
		return fmt.Errorf("path is empty")
	}
	return setOnMapping(root, path, value)
}

func setOnMapping(node *yaml.Node, path []string, value *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%s is not a mapping", strings.Join(path, "."))
	}

	key := path[0]
	idx := mappingKeyIndex(node, key)
	last := len(path) == 1

	if last {
		if isNullNode(value) {
			if idx >= 0 {
				node.Content = append(node.Content[:idx], node.Content[idx+2:]...)
			}
			return nil
		}
		if idx >= 0 {
			node.Content[idx+1] = value
			return nil
		}
		node.Content = append(node.Content, scalarKey(key), value)
		return nil
	}

	if isNullNode(value) && idx < 0 {
		return nil
	}

	var child *yaml.Node
	if idx < 0 {
		child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		node.Content = append(node.Content, scalarKey(key), child)
	} else {
		child = node.Content[idx+1]
		if child.Kind != yaml.MappingNode {
			return fmt.Errorf("cannot set %s: %s is not a mapping", strings.Join(path, "."), key)
		}
	}
	return setOnMapping(child, path[1:], value)
}

func mappingKeyIndex(node *yaml.Node, key string) int {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func scalarKey(key string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
}

func isNullNode(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}
