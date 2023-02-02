package tokenmgr

import "github.com/SwissDataScienceCenter/renku-gateway-v2/internal/models"

type AccessTokenReader interface {
	Read(tokenID string) (models.AccessToken, error)
}

type AccessTokenWriter interface {
	Write(models.AccessToken) error
}

type TokenRemover interface {
	Remove(tokenID string) error
}

type RefreshTokenReader interface {
	Read(tokenID string) (models.RefreshToken, error)
}

type RefreshTokenWriter interface {
	Write(models.RefreshToken) error
}

type AccessTokenReaderWriterRemover interface {
	AccessTokenReader
	AccessTokenWriter
	TokenRemover
}

type RefreshTokenReaderWriterRemover interface {
	RefreshTokenReader
	RefreshTokenWriter
	TokenRemover
}
