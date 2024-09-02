package models

type Encryptor interface {
	Encrypt(value string) (encrypted string, err error)
	Decrypt(value string) (decrypted string, err error)
}
