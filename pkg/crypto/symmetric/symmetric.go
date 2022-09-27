package symmetric

import (
	"bytes"
	rand "crypto/rand"

	"shlyuz/pkg/crypto/rc6"
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

// pad a message
//
//	See: https://github.com/go-web/tokenizer/blob/master/pkcs7.go
//	This assumes the message is perfect. It will trigger a crash if it recieves invalid data
func pad(message []byte) []byte {
	// TODO: Handle invalid data here
	n := 16 - ((len(message)) % 16)
	paddedBytes := make([]byte, len(message)+n)
	copy(paddedBytes, message)
	copy(paddedBytes[len(message):], bytes.Repeat([]byte{byte(n)}, n))
	return paddedBytes
}

// Remove padding from a given message
//
//	This assumes the message is perfect. It will trigger a crash if it recieves invalid data.
func unpad(message []byte) []byte {
	size := message[len(message)-1]
	x := int(size)
	// TODO: handle invalid padding here
	return message[:len(message)-x]
}

// Encrypts a message, given a plaintext
//
//	Returns a SymmetricMessage. Caller should check to see if Key is populat
func Encrypt(plaintext []byte) SymmetricMessage {
	var EncryptedMessage SymmetricMessage
	var paddedPlaintext []byte
	var paddedChunk []byte

	if (len(plaintext) < 16) || (len(plaintext)%4 != 0) {
		paddedPlaintext = pad(plaintext)
	} else {
		paddedPlaintext = plaintext
	}

	EncryptedMessage.Key = generateKey()

	cipher := rc6.NewCipher(EncryptedMessage.Key)
	// Chunking logic, appends to EncryptedMessage.Message every 16 bytes
	//   APPEND to EncryptedMessage.Message since this library won't loop for you. Loop across every 16 bytes of paddedPlaintext (paddedChunk)
	for i := 0; i < len(paddedPlaintext); i = i + 16 {
		j := i + 16
		paddedChunk = paddedPlaintext[i:j]
		cipher.Encrypt(paddedChunk, paddedPlaintext[i:j])
		EncryptedMessage.Message = append(EncryptedMessage.Message[:], paddedChunk[:]...)
	}

	EncryptedMessage.IsEncrypted = true

	return EncryptedMessage
}

// Decrypts a message, given an encrypted text and a decryption key
//
//	Returns a SymmetricMessage
func Decrypt(encryptedText []byte, decryptionKey []byte) SymmetricMessage {
	var DecryptedMessage SymmetricMessage
	var unpaddedText []byte
	var unpaddedChunk []byte

	// TODO: We should REALLY check if `!(len(decryptionKey) % 4) != 0` here, and safely handle that before continuing.
	DecryptedMessage.Key = decryptionKey
	cipher := rc6.NewCipher(decryptionKey)

	if (len(encryptedText) < 16) || (len(encryptedText)%4 != 0) {
		unpaddedText = unpad(encryptedText)
	} else {
		unpaddedText = encryptedText
	}
	// Chunking logic, appends to EncryptedMessage.Message every 16 bytes
	//   APPEND to EncryptedMessage.Message since this library won't loop for you. Loop across every 16 bytes of paddedPlaintext (paddedChunk)
	for i := 0; i < len(unpaddedText); i = i + 16 {
		j := i + 16
		unpaddedChunk = unpaddedText[i:j]
		cipher.Decrypt(unpaddedChunk, unpaddedText[i:j])
		DecryptedMessage.Message = append(DecryptedMessage.Message[:], unpaddedChunk[:]...)
	}
	//   APPEND to DecryptedMessage.Message since this library won't loop for you. You need to loop across every 16 bytes of plaintext
	DecryptedMessage.IsEncrypted = false

	return DecryptedMessage
}
