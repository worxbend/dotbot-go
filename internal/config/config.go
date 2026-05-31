package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Task map[string]any

func Read(paths []string) ([]Task, error) {
	var tasks []Task
	for _, path := range paths {
		fileTasks, err := readOne(path)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, fileTasks...)
	}
	return tasks, nil
}

func readOne(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}
	var raw any
	if filepath.Ext(path) == ".json" {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("could not read config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("could not read config file: %w", err)
		}
	}
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("configuration file must be a list of tasks")
	}
	tasks := make([]Task, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("configuration task must be a mapping")
		}
		tasks = append(tasks, Task(m))
	}
	return tasks, nil
}
