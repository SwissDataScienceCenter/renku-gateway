package models

import (
	"encoding/json"
)

type SerializableMap map[string]string

func (s SerializableMap) MarshalBinary() (data []byte, err error) {
	return json.Marshal(map[string]string(s))
}

func (s *SerializableMap) UnmarshalBinary(data []byte) error {
	var res map[string]string
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = res
	return nil
}

func (s SerializableMap) MarshalText() (data []byte, err error) {
	return s.MarshalBinary()
}

func (s *SerializableMap) UnmarshalText(data []byte) error {
	return s.UnmarshalBinary(data)
}
