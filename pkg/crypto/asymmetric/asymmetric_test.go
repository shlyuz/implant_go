package asymmetric

import (
	"bytes"
	"testing"
)

type KeyPair struct {
	PublicKey  *[32]byte
	PrivateKey *[32]byte
}

// Bytes in this test are bytes that tend to just randomly break things such as newlines, nulls, NOPs, INT3, etc
var breakingBytesTest = []struct {
	message []byte
}{
	{
		// Nop Sled
		[]byte{144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144, 144},
	},
	{
		// INT3 Sled
		[]byte{204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204, 204},
	},
	{
		// CRLF Sled
		[]byte{10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13, 10, 13},
	},
	{
		// LF Sled
		[]byte{13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13, 13},
	},
	{
		// Null Sled
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
}

func TestAsymmetricNalKeyGeneration(t *testing.T) {
	t.Parallel()
	_, _, err := GenerateKeypair()
	if err != nil {
		t.Log("[FAIL] NaclKeyGeneration Failed")
		t.Error("Error: ", err)
	}
}

func TestAsymmetricNaclEncryptionAndDecryption(t *testing.T) {
	t.Parallel()
	for _, testcase := range breakingBytesTest {
		var err error
		var secretMessage AsymmetricBox
		secretMessage = testcase.message

		AliceKeyPair := new(KeyPair)
		AliceKeyPair.PublicKey, AliceKeyPair.PrivateKey, err = GenerateKeypair()
		if err != nil {
			t.Log("[FAIL] Couldn't generate Alice keypair")
			t.Error("Error: ", err)
		}

		BobKeyPair := new(KeyPair)
		BobKeyPair.PublicKey, BobKeyPair.PrivateKey, err = GenerateKeypair()
		if err != nil {
			t.Log("[FAIL] Couldn't generate Bob keypair")
			t.Error("Error: ", err)
		}

		// t.Log("Testing Message from Alice to Bob")
		AliceEncryptedMessage := Encrypt(secretMessage, BobKeyPair.PublicKey, AliceKeyPair.PrivateKey)
		if bytes.Equal(AliceEncryptedMessage, secretMessage) {
			t.Log("[FAIL] Alice Message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}

		BobEncryptedMessage := Encrypt(secretMessage, AliceKeyPair.PublicKey, BobKeyPair.PrivateKey)
		if bytes.Equal(BobEncryptedMessage, secretMessage) {
			t.Log("[FAIL] Bob message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}
	}
}
