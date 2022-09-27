package symmetric

import (
	"math/rand"

	rc6 "github.com/CampNowhere/golang-rc6"
)

type SymmetricMessage struct {
	Message     []byte
	Key         []byte
	IsEncrypted bool
}

func generateKey() []byte {
	key := make([]byte, 16)
	rand.Read(key)
	return key
}

// Encrypts a message, given a plaintext
//
//	Returns a SymmetricMessage
func Encrypt(plaintext []byte) SymmetricMessage {
	var EncryptedMessage SymmetricMessage

	key := generateKey()
	EncryptedMessage.Key = key

	cipher := rc6.NewCipher(EncryptedMessage.Key)
	// TODO: Check if `(len(plaintext) %4 != 0`. If not, add padding.
	//   APPEND to EncryptedMessage.Message since this library won't loop for you. You need to loop across every 16 bytes of plaintext
	cipher.Encrypt(EncryptedMessage.Message, plaintext)
	EncryptedMessage.IsEncrypted = true

	return EncryptedMessage
}

// Decrypts a message, given an encrypted text and a decryption key
//
//	Returns a SymmetricMessage
func Decrypt(encryptedText []byte, decryptionKey []byte) SymmetricMessage {
	var DecryptedMessage SymmetricMessage

	// TODO: We should REALLY check if `!(len(decryptionKey) % 4) != 0` here, and safely handle that before continuing.
	DecryptedMessage.Key = decryptionKey

	cipher := rc6.NewCipher(decryptionKey)
	//   APPEND to DecryptedMessage.Message since this library won't loop for you. You need to loop across every 16 bytes of plaintext
	cipher.Decrypt(DecryptedMessage.Message, encryptedText)
	DecryptedMessage.IsEncrypted = false

	return DecryptedMessage
}
