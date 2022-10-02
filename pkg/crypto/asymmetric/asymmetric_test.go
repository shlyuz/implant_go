package asymmetric

import (
	"bytes"
	"testing"
)

// Bytes in this test are bytes that tend to just randomly break things such as newlines, nulls, NOPs, INT3, etc
var breakingBytesTest = []struct {
	message []byte
}{
	{
		[]byte("sub 16 len str"),
	},
	{
		[]byte("16 len str xxxxx"),
	},
	{
		[]byte("Sub 32 len string, but > 16"),
	},
	{
		[]byte("32 length string is placed here."),
	},
	{
		[]byte("string len >32 but also less than 48"),
	},
	{
		[]byte("string len >48, and is also mod % 4, so no padding"),
	},
	{
		[]byte("okay this string should be split up into exactly 4 chunks no pad"),
	},
	{
		[]byte("Finally this is going to be a very long string over 64 bytes in length, with padding."),
	},
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
	_, err := GenerateKeypair()
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
		secretMessage.Message = testcase.message

		AliceKeyPair := new(AsymmetricKeyPair)
		*AliceKeyPair, err = GenerateKeypair()
		if err != nil {
			t.Log("[FAIL] Couldn't generate Alice keypair")
			t.Error("Error: ", err)
		}

		BobKeyPair := new(AsymmetricKeyPair)
		*BobKeyPair, err = GenerateKeypair()
		if err != nil {
			t.Log("[FAIL] Couldn't generate Bob keypair")
			t.Error("Error: ", err)
		}

		// t.Log("Testing Message from Alice to Bob")
		AliceEncryptedBox := Encrypt(secretMessage.Message, BobKeyPair.PubKey, AliceKeyPair.PrivKey)
		if bytes.Equal(AliceEncryptedBox.Message, testcase.message) {
			t.Log("[FAIL] Alice Message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}

		BobEncryptedBox := Encrypt(secretMessage.Message, AliceKeyPair.PubKey, BobKeyPair.PrivKey)
		if bytes.Equal(BobEncryptedBox.Message, testcase.message) {
			t.Log("[FAIL] Bob message encryption failed")
			t.Error("Testcase: ", testcase.message)
		}

		AliceDecryptedMessage, decryptionSuccess := Decrypt(AliceEncryptedBox, AliceKeyPair.PubKey, BobKeyPair.PrivKey)
		if !(decryptionSuccess) || !bytes.Equal(AliceDecryptedMessage.Message, testcase.message) {
			t.Log("[FAIL] Bob failed to decrypt Alice's message")
			t.Error("Testcase: ", testcase.message)
		}

		BobDecryptedMessage, decryptionSuccess := Decrypt(BobEncryptedBox, BobKeyPair.PubKey, AliceKeyPair.PrivKey)
		if !(decryptionSuccess) || !bytes.Equal(BobDecryptedMessage.Message, testcase.message) {
			t.Log("[FAIL] Bob failed to decrypt Bob's message")
			t.Error("Testcase: ", testcase.message)
		}
	}
	t.Log("[PASS]")
}
