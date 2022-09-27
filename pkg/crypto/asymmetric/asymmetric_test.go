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
		secretMessage.message = testcase.message

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
		AliceEncryptedBox := Encrypt(secretMessage.message, BobKeyPair.PublicKey, AliceKeyPair.PrivateKey)
		if bytes.Equal(AliceEncryptedBox.message, testcase.message) {
			t.Log("[FAIL] Alice Message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}

		BobEncryptedBox := Encrypt(secretMessage.message, AliceKeyPair.PublicKey, BobKeyPair.PrivateKey)
		if bytes.Equal(BobEncryptedBox.message, testcase.message) {
			t.Log("[FAIL] Bob message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}

		AliceDecryptedMessage, decryptionSuccess := Decrypt(*AliceEncryptedBox, AliceKeyPair.PublicKey, BobKeyPair.PrivateKey)
		if !(decryptionSuccess) || !bytes.Equal(AliceDecryptedMessage.message, testcase.message) {
			t.Log("[FAIL] Bob failed to decrypt Alice's message")
			t.Error("Testcase: ", testcase.message)
		}

		BobDecryptedMessage, decryptionSuccess := Decrypt(*BobEncryptedBox, BobKeyPair.PublicKey, AliceKeyPair.PrivateKey)
		if !(decryptionSuccess) || !bytes.Equal(BobDecryptedMessage.message, testcase.message) {
			t.Log("[FAIL] Bob failed to decrypt Bob's message")
			t.Error("Testcase: ", testcase.message)
		}
		t.Log("[PASS]")
	}
}
