package models

import (
	"encoding/json"
)

type SerializableMap map[string]string

func (s SerializableMap) MarshalBinary() (data []byte, err error) {
	if s == nil {
		s = SerializableMap(map[string]string{})
	}
	return json.Marshal(map[string]string(s))
}

func (s *SerializableMap) UnmarshalBinary(data []byte) error {
	m := new(map[string]string)
	err := json.Unmarshal(data, m)
	if err != nil {
		return err
	}
	*s = SerializableMap(*m)
	return nil
}

func (s SerializableMap) MarshalText() (data []byte, err error) {
	return s.MarshalBinary()
}

func (s *SerializableMap) UnmarshalText(data []byte) error {
	return s.UnmarshalBinary(data)
}

func NewSerializableMap() SerializableMap {
	return SerializableMap(map[string]string{})
}
