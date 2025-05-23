package symmetric

import (
	"bytes"
	"crypto/hmac"
	rand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"

	rc6 "shlyuz/pkg/crypto/rc6"
)

type SymmetricMessage struct {
	Message     []byte
	Key         []byte
	IsEncrypted bool
}

func generateKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// Add padding to a given message
//
//	See: https://github.com/go-web/tokenizer/blob/master/pkcs7.go
//	This assumes the message is perfect. It will trigger a crash if it recieves invalid data
func pad(message []byte) []byte {
	n := 16 - (len(message) % 16)
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
	if x == 0 || x > 16 {
		return message
	}
	return message[:len(message)-x]
}

// Encrypts a message, given a plaintext. Returns a SymmetricMessage. Caller should check to see if Key is populated
//
// @param plaintext: A plaintext byte array with the message to be encrypted
func Encrypt(plaintext []byte) SymmetricMessage {
	var EncryptedMessage SymmetricMessage
	EncryptedMessage.Key = generateKey()
	rc6Key := EncryptedMessage.Key[:16]
	hmacKey := EncryptedMessage.Key[16:]

	// Generate IV
	iv := make([]byte, 16)
	rand.Read(iv)

	paddedPlaintext := pad(plaintext)
	ciphertext := make([]byte, len(paddedPlaintext))
	cipher := rc6.NewCipher(rc6Key)
	prevCiphertextBlock := iv

	for i := 0; i < len(paddedPlaintext); i += 16 {
		block := paddedPlaintext[i : i+16]
		xorBlock := make([]byte, 16)
		for j := 0; j < 16; j++ {
			xorBlock[j] = block[j] ^ prevCiphertextBlock[j]
		}
		encryptedBlock := make([]byte, 16)
		cipher.Encrypt(encryptedBlock, xorBlock)
		copy(ciphertext[i:i+16], encryptedBlock)
		prevCiphertextBlock = encryptedBlock
	}

	// Calculate HMAC
	dataToMac := append(iv, ciphertext...)
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(dataToMac)
	hmacValue := mac.Sum(nil)

	EncryptedMessage.Message = append(append(iv, ciphertext...), hmacValue...)
	EncryptedMessage.IsEncrypted = true

	return EncryptedMessage
}

// Decrypts a message, given an encrypted text and a decryption key. Returns a SymmetricMessage on error or success. SymmetricMessage.IsEncrypted will be true upong failure.
//
// @param encryptedText: A [][byte of RC6 encrypted text to decrypt
// @param decryptionKey: A [32]byte key to be used for decryption
func Decrypt(encryptedText []byte, decryptionKey []byte) SymmetricMessage {
	var DecryptedMessage SymmetricMessage
	DecryptedMessage.Key = decryptionKey
	DecryptedMessage.IsEncrypted = true // Default to true, set to false on success

	if len(decryptionKey) != 32 {
		DecryptedMessage.Message = encryptedText // Or some error message
		return DecryptedMessage
	}

	rc6Key := decryptionKey[:16]
	hmacKey := decryptionKey[16:]

	// Minimum length: IV (16) + 1 block Ciphertext (16) + HMAC (32) = 64
	if len(encryptedText) < 64 {
		DecryptedMessage.Message = encryptedText // Or some error message indicating too short
		return DecryptedMessage
	}

	iv := encryptedText[:16]
	hmacValue := encryptedText[len(encryptedText)-32:]
	ciphertext := encryptedText[16 : len(encryptedText)-32]

	if len(ciphertext)%16 != 0 {
		// Ciphertext must be a multiple of block size
		DecryptedMessage.Message = encryptedText // Or some error
		return DecryptedMessage
	}

	// Verify HMAC
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(append(iv, ciphertext...))
	expectedMAC := mac.Sum(nil)

	if subtle.ConstantTimeCompare(hmacValue, expectedMAC) != 1 {
		DecryptedMessage.Message = encryptedText // Or some error message
		return DecryptedMessage
	}

	// CBC Decryption
	decryptedPaddedPlaintext := make([]byte, len(ciphertext))
	cipher := rc6.NewCipher(rc6Key)
	prevCiphertextBlock := iv

	for i := 0; i < len(ciphertext); i += 16 {
		block := ciphertext[i : i+16]
		decryptedBlock := make([]byte, 16)
		cipher.Decrypt(decryptedBlock, block)

		xorBlock := make([]byte, 16)
		for j := 0; j < 16; j++ {
			xorBlock[j] = decryptedBlock[j] ^ prevCiphertextBlock[j]
		}
		copy(decryptedPaddedPlaintext[i:i+16], xorBlock)
		prevCiphertextBlock = block
	}

	DecryptedMessage.Message = unpad(decryptedPaddedPlaintext)
	DecryptedMessage.IsEncrypted = false

	return DecryptedMessage
}
