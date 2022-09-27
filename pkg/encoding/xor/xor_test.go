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
