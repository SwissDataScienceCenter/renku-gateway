package models

import (
	"encoding/json"

	"github.com/wk8/go-ordered-map/v2"
)

type SerializableOrderedMap struct {
	*orderedmap.OrderedMap[string, string]
}

func (s SerializableOrderedMap) MarshalBinary() (data []byte, err error) {
	if s.OrderedMap == nil {
		s.OrderedMap = orderedmap.New[string, string]()
	}
	return json.Marshal(s.OrderedMap)
}

func (s *SerializableOrderedMap) UnmarshalBinary(data []byte) error {
	if s.OrderedMap == nil {
		s.OrderedMap = orderedmap.New[string, string]()
	}
	return json.Unmarshal(data, s)
}

func (s SerializableOrderedMap) MarshalText() (data []byte, err error) {
	if s.OrderedMap == nil {
		s.OrderedMap = orderedmap.New[string, string]()
	}
	return json.Marshal(s)
}

func (s *SerializableOrderedMap) UnmarshalText(data []byte) error {
	return s.UnmarshalBinary(data)
}

func NewSerializableOrderedMap() SerializableOrderedMap {
	data := orderedmap.New[string, string]()
	return SerializableOrderedMap{data}
}
