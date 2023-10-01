package models

type SerializableBool bool

func (s SerializableBool) MarshalBinary() (data []byte, err error) {
	if s {
		return []byte{byte(1)}, nil
	}
	return []byte{byte(0)}, nil
}

func (s *SerializableBool) UnmarshalBinary(data []byte) error {
	if data[0] == byte(1) {
		*s = true
	}
	*s = false
	return nil
}

func (s SerializableBool) MarshalText() (data []byte, err error) {
	if s {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

func (s *SerializableBool) UnmarshalText(data []byte) error {
	if string(data) == "true" {
		*s = true
	}
	*s = false
	return nil
}
