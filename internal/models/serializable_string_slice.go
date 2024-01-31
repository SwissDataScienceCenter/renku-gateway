package models

import "encoding/json"

// SerializableStringSlice is a string slice that can be marshalled to a json representation
// of a string slice and also unmarshalled back into a list of strings from a json string.
type SerializableStringSlice []string

func (s SerializableStringSlice) MarshalBinary() (data []byte, err error) {
	return json.Marshal(s)
}

func (s *SerializableStringSlice) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s SerializableStringSlice) MarshalText() (data []byte, err error) {
	return json.Marshal(s)
}

func (s *SerializableStringSlice) UnmarshalText(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s SerializableStringSlice) MarshalJSON() (data []byte, err error) {
	// Marshal to JSON same as []string
	return json.Marshal([]string(s))
}

func (s *SerializableStringSlice) UnmarshalJSON(data []byte) error {
	// Unmarshal to json same as []string
	var res []string
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err
	}
	*s = res
	return nil
}
