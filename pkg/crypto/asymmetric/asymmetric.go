package asymmetric

import (
	"crypto/rand"
	"log"

	"shlyuz/pkg/utils/logging"

	"github.com/keys-pub/keys"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
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
	edKey := keys.NewX25519KeyFromPrivateKey(privKey)
	pubKey := edKey.PublicKey().Bytes()
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

func DecryptSealed(encBox AsymmetricBox, decryptionKey PrivateKey) (*AsymmetricBox, bool) {
	decryptedSealedBox := new(AsymmetricBox)
	decryptedSealedBox.IV = encBox.IV
	var output []byte
	output, boolSuccess := secretbox.Open(output, encBox.Message, encBox.IV, decryptionKey)
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
	output = secretbox.Seal(output, message, nonce, peerPublicKey)
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
