package models

type OauthTokenType string

const AccessTokenType OauthTokenType = "AccessToken"
const RefreshTokenType OauthTokenType = "RefreshToken"

func (o OauthTokenType) MarshalText() (data []byte, err error) {
	return []byte(o), nil
}

func (o OauthTokenType) MarshalBinary() (data []byte, err error) {
	return []byte(o), nil
}

func (o *OauthTokenType) UnmarshalText(data []byte) error {
	*o = OauthTokenType(string(data))
	return nil
}

func (o *OauthTokenType) UnmarshalBinary(data []byte) error {
	*o = OauthTokenType(string(data))
	return nil
}
