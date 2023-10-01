package models

import "strconv"

type SessionType int

const (
	Default SessionType = iota
	Cli
	Kg
)

func (s SessionType) String() string {
	return [...]string{"Default", "Cli", "Kg"}[s]
}

func NewSessionType(id int) SessionType {
	return [...]SessionType{Default, Cli, Kg}[id]
}

func (s SessionType) MarshalBinary() (data []byte, err error) {
	return s.MarshalText()
}

func (s *SessionType) UnmarshalBinary(data []byte) error {
	return s.UnmarshalText(data)
}

func (s SessionType) MarshalText() (data []byte, err error) {
	return []byte(strconv.Itoa(int(s))), nil
}

func (s *SessionType) UnmarshalText(data []byte) error {
	res, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	*s = [...]SessionType{Default, Cli, Kg}[res]
	return nil
}
