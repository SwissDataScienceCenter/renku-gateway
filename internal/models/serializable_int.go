package models

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

type SerializableInt int

func (s SerializableInt) MarshalBinary() (data []byte, err error) {
	output := make([]byte, 8)
	binary.PutVarint(output, int64(s))
	return output, nil
}

func (s *SerializableInt) UnmarshalBinary(data []byte) error {
	input := bytes.NewReader(data)
	res, err := binary.ReadVarint(input)
	if err != nil {
		return err
	}
	*s = SerializableInt(res)
	return nil
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
