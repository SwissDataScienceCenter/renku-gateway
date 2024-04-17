package models

import (
	"encoding/json"
	"strconv"
)

type SerializableInt int

func (s SerializableInt) MarshalBinary() (data []byte, err error) {
	return json.Marshal(s)
}

func (s *SerializableInt) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s SerializableInt) MarshalText() (data []byte, err error) {
	return []byte(strconv.Itoa(int(s))), nil
}

func (s *SerializableInt) UnmarshalText(data []byte) error {
	val, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	*s = SerializableInt(val)
	return nil
}
