package tokenmgr

type RefreshTokenManager struct {
	refreshStore RefreshTokenReaderWriterRemover
	accessStore  AccessTokenReaderWriterRemover
}
