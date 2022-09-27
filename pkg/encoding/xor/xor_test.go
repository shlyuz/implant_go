package xor

import (
	"bytes"
	"math/rand"
	"testing"
)

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

func genRandomKey(t *testing.T) int {
	t.Helper()
	return rand.Intn(254)
}

func TestXorEncoder(t *testing.T) {
	t.Parallel()
	for _, testcase := range breakingBytesTest {
		xorKey := genRandomKey(t)
		encodedMessage := XorMessage(testcase.message, xorKey)
		if bytes.Equal(encodedMessage, testcase.message) {
			t.Log("[FAIL] Xor encoder failed to encode the given message")
			t.Log("Testcase: ", testcase.message)
			t.Log("Generated Key: ", xorKey)
			t.Error("Encoding failed")
		}

		decodedMessage := XorMessage(encodedMessage, xorKey)
		if !bytes.Equal(decodedMessage, testcase.message) {
			t.Log("[FAIL] Xor encoder failed to decode the given message")
			t.Log("Testcase: ", testcase.message)
			t.Log("Generated Key: ", xorKey)
			t.Error("Encoding failed")
		}
	}
	t.Log("[PASS]")
}
