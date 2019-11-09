package mapbuilder

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	Separator = "."
)

func NewMapBuilder() *MapBuilder {
	return &MapBuilder{}
}

func NewMapBuilderWithInit(body map[string]interface{}) *MapBuilder {
	return &MapBuilder{
		body: body,
	}
}

// MapBuilder creates body builder
type MapBuilder struct {
	body map[string]interface{}
}

// SetMap sets a new map to the body. This will remove any existing values in the body
func (b *MapBuilder) SetMap(body map[string]interface{}) {
	b.body = body
}

// GetMap returns the body as a map[string]interface{}
func (b MapBuilder) GetMap() map[string]interface{} {
	return b.body
}

// MarshalJSON returns the body as json
func (b MapBuilder) MarshalJSON() ([]byte, error) {
	if b.body == nil {
		return nil, errors.New("body is uninitialized")
	}
	return json.Marshal(b.body)
}

// Set sets a value to a give dot notation path
func (b *MapBuilder) Set(path string, value interface{}) error {
	if b.body == nil {
		b.body = make(map[string]interface{})
	}
	keys := strings.Split(path, Separator)

	currentMap := b.body

	lastIndex := len(keys) - 1

	for i, key := range keys {
		if key != "" {
			if i != lastIndex {
				if _, ok := currentMap[key]; !ok {
					currentMap[key] = make(map[string]interface{})
				}
				currentMap = currentMap[key].(map[string]interface{})
			} else {
				currentMap[key] = value
			}
		}
	}

	return nil
}
