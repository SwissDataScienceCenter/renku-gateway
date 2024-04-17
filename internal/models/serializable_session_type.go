package models

type SessionType string

const Default SessionType = "Default"
const Device SessionType = "Device"

func (s SessionType) MarshalText() (data []byte, err error) {
	return []byte(s), nil
}

func (s SessionType) MarshalBinary() (data []byte, err error) {
	return []byte(s), nil
}

func (s *SessionType) UnmarshalText(data []byte) error {
	*s = SessionType(string(data))
	return nil
}

func (s *SessionType) UnmarshalBinary(data []byte) error {
	*s = SessionType(string(data))
	return nil
}
