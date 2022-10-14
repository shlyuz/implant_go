package asymmetric

import (
	"crypto/rand"
	"io"
	"log"

	"shlyuz/pkg/utils/logging"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

type Nonce = *[24]byte

type AsymmetricBox struct {
	Message []byte
	IV      Nonce
}

type AsymmetricKeyPair struct {
	PubKey  PublicKey
	PrivKey PrivateKey
}

type PublicKey = *[32]byte
type PrivateKey = *[32]byte

func PubFromPriv(privKey PrivateKey) *PublicKey {
	pubKey := new([32]byte)
	curve25519.ScalarBaseMult(pubKey, privKey)
	publicKey := (*[32]byte)(pubKey)
	return &publicKey
}

func generateNonce() Nonce {
	log.SetPrefix(logging.GetLogPrefix())
	var nonce Nonce
	randomNonce := make([]byte, 24)
	_, err := rand.Read(randomNonce)
	if err != nil {
		log.Panicln("failed to generate nonce ", err)
	}
	nonce = (*[24]byte)(randomNonce)
	return nonce
}

// Generates a nacl box keypair
func GenerateKeypair() (AsymmetricKeyPair, error) {
	log.SetPrefix(logging.GetLogPrefix())
	var keyPair AsymmetricKeyPair

	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	keyPair.PubKey = pubKey
	keyPair.PrivKey = privKey
	if err != nil {
		log.Println("failed to generate keypair: ", err)
	}
	return keyPair, err
}

func DecryptSealed(encBox AsymmetricBox, decryptionKey PrivateKey, pubKey PublicKey) (*AsymmetricBox, bool) {
	decryptedSealedBox := new(AsymmetricBox)
	decryptedSealedBox.IV = encBox.IV
	var output []byte
	output, boolSuccess := box.OpenAnonymous(output, encBox.Message, pubKey, decryptionKey)
	if !boolSuccess {
		log.Println("failed to open sealed box, received: ", boolSuccess)
		return &encBox, boolSuccess
	}
	decryptedSealedBox.Message = output
	return decryptedSealedBox, true
}

// Returns an Asymmetric box with the decrypted contents
//
// @param encBox: A pointer to the AsymmetricBox containing the message to decrypt
// @param peersPublicKey: A pointer to the PublicKey we open the box with
// @param privateKey: A pointer to the PrivateKey we open the box with
func Decrypt(encBox AsymmetricBox, peersPublicKey PublicKey, privateKey PrivateKey) (*AsymmetricBox, bool) {
	log.SetPrefix(logging.GetLogPrefix())
	var output []byte
	decryptedBox := new(AsymmetricBox)
	output, boolSuccess := box.Open(output, encBox.Message, encBox.IV, peersPublicKey, privateKey)
	if !boolSuccess {
		log.Println("failed to open secret box, received: ", boolSuccess)
		return &encBox, boolSuccess
	}
	decryptedBox.IV = encBox.IV
	decryptedBox.Message = output
	return decryptedBox, boolSuccess
}

func EncryptSealed(message []byte, peerPublicKey PublicKey) AsymmetricBox {
	encryptedSealedBox := new(AsymmetricBox)
	nonce := generateNonce()
	encryptedSealedBox.IV = nonce
	var output []byte
	var throwAway io.Reader
	output, err := box.SealAnonymous(output, message, peerPublicKey, throwAway)
	if err != nil {
		log.Fatalln("failed to generate init message: ", err)
	}
	_ = throwAway
	encryptedSealedBox.Message = output
	return *encryptedSealedBox
}

// Retruns the encrypted output from an Aymmetric box
//
// @param message: A byte array, but initalized as an AsymmetricBox
// @param peersPublicKey: A pointer to the PublicKey we open the box with
// @param privateKey: A pointer to the PrivateKey we open the box with
func Encrypt(message []byte, peersPublicKey PublicKey, privateKey PrivateKey) AsymmetricBox {
	encryptedBox := new(AsymmetricBox)
	nonce := generateNonce()
	encryptedBox.IV = nonce
	var output []byte
	output = box.Seal(output, message, nonce, peersPublicKey, privateKey)
	encryptedBox.Message = output
	return *encryptedBox
}
