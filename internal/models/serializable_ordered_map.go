package models

import (
	"encoding/json"

	orderedmap "github.com/wk8/go-ordered-map/v2"
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
	return json.Unmarshal(data, &s.OrderedMap)
}

func (s SerializableOrderedMap) MarshalText() (data []byte, err error) {
	// if s.OrderedMap == nil {
	// 	s.OrderedMap = orderedmap.New[string, string]()
	// }
	// return json.Marshal(s)
	return s.MarshalBinary()
}

func (s *SerializableOrderedMap) UnmarshalText(data []byte) error {
	// return s.UnmarshalBinary(data)
	return s.UnmarshalBinary(data)
}

func NewSerializableOrderedMap() SerializableOrderedMap {
	data := orderedmap.New[string, string]()
	return SerializableOrderedMap{data}
}
