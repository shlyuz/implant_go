package symmetric

import (
	"bytes"
	"testing"
)

var tests = []struct {
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
}

type TestMessageType interface {
	getMessageStruct() SymmetricMessage
}

func (s SymmetricMessage) getMessageStruct() SymmetricMessage {
	return s
}

func attemptEncryption(MessageToEncrypt []byte) SymmetricMessage {
	return Encrypt(MessageToEncrypt)
}

func attemptDecryption(msg TestMessageType) SymmetricMessage {
	encryptedMsg := msg.getMessageStruct()
	return Decrypt(encryptedMsg.Message, encryptedMsg.Key)
}

func TestSymmetricRc6EncryptionAndDecryption(t *testing.T) {
	t.Parallel()
	for _, testcase := range tests {
		EncryptionTestMessageBytes := make([]byte, len(testcase.message))
		copy(EncryptionTestMessageBytes, testcase.message)
		if !bytes.Equal(EncryptionTestMessageBytes, testcase.message) {
			t.Log("Setting the plaintext message failed")
			t.Errorf("%s != %s", EncryptionTestMessageBytes, testcase.message)
		}
		SymmetricEncryptionTestCase := attemptEncryption(EncryptionTestMessageBytes)

		if bytes.Equal(SymmetricEncryptionTestCase.Message, testcase.message) {
			t.Log("[FAIL] Encryption attempt failed")
			t.Log("Generated key for encryption testcase: ", SymmetricEncryptionTestCase.Key)
			t.Log("IsEncrypted value: ", SymmetricEncryptionTestCase.IsEncrypted)
			t.Error("Testcase: ", testcase.message)
		} else {
			t.Log("[PASS] Encryption for Testcase: ", string(testcase.message))
		}

		SymmetricEncryptionTestCase = SymmetricEncryptionTestCase.getMessageStruct()
		SymmetricEncryptionTestCase = attemptDecryption(SymmetricEncryptionTestCase)

		if !bytes.Equal(SymmetricEncryptionTestCase.Message, testcase.message) {
			t.Log("[FAIL] Decryption attempt failed")
			t.Log("Retrieved key for decryption testcase: ", SymmetricEncryptionTestCase.Key)
			t.Log("IsEncrypted value: ", SymmetricEncryptionTestCase.IsEncrypted)
			t.Log("Equal?: ", bytes.Equal(SymmetricEncryptionTestCase.Message, testcase.message))
			t.Log("Decrypted message: ", SymmetricEncryptionTestCase.Message)
			t.Error("Testcase: ", string(testcase.message))
		} else {
			t.Log("[PASS] Decryption for Testcase: ", string(testcase.message))
		}
	}
}
