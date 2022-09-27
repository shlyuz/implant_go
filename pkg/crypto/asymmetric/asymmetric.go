package asymmetric

import (
	"crypto/rand"
	"log"

	"golang.org/x/crypto/nacl/box"
)

type Nonce = *[24]byte
type AsymmetricBox struct {
	message []byte
	iv      Nonce
}
type PublicKey = *[32]byte
type PrivateKey = *[32]byte

func generateNonce() Nonce {
	var nonce Nonce
	randomNonce := make([]byte, 24)
	_, err := rand.Read(randomNonce)
	if err != nil {
		log.Panicln("Failed to generate nonce ", err)
	}
	nonce = (*[24]byte)(randomNonce)
	return nonce
}

// Generates a nacl box keypair
func GenerateKeypair() (PublicKey, PrivateKey, error) {
	var PubKey PublicKey
	var PrivKey PrivateKey
	PubKey, PrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Println("Failed to generate keypair: ", err)
	}
	return PubKey, PrivKey, err
}

// Returns an Asymmetric box with the decrypted contents
//
// @param encBox: A pointer to the AsymmetricBox containing the message to decrypt
// @param peersPublicKey: A pointer to the PublicKey we open the box with
// @param privateKey: A pointer to the PrivateKey we open the box with
func Decrypt(encBox AsymmetricBox, peersPublicKey PublicKey, privateKey PrivateKey) (*AsymmetricBox, bool) {
	var output []byte
	decryptedBox := new(AsymmetricBox)
	output, boolSuccess := box.Open(output, encBox.message, encBox.iv, peersPublicKey, privateKey)
	if !boolSuccess {
		log.Println("Failed to open secret box, received: ", boolSuccess)
	}
	decryptedBox.iv = encBox.iv
	decryptedBox.message = output
	return decryptedBox, boolSuccess
}

// Retruns the encrypted output from an Aymmetric box
//
// @param message: A byte array, but initalized as an AsymmetricBox
// @param peersPublicKey: A pointer to the PublicKey we open the box with
// @param privateKey: A pointer to the PrivateKey we open the box with
func Encrypt(message []byte, peersPublicKey PublicKey, privateKey PrivateKey) *AsymmetricBox {
	encryptedBox := new(AsymmetricBox)
	nonce := generateNonce()
	encryptedBox.iv = nonce
	var output []byte
	output = box.Seal(output, message, nonce, peersPublicKey, privateKey)
	encryptedBox.message = output
	return encryptedBox
}
