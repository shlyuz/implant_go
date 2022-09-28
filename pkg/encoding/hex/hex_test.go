package hex

import (
	"bytes"
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

func TestHexEncoder(t *testing.T) {
	t.Parallel()
	for _, testcase := range breakingBytesTest {
		encodedMessage := Encode(testcase.message)
		if bytes.Equal(encodedMessage, testcase.message) {
			t.Log("[FAIL] Xor encoder failed to encode the given message")
			t.Log("Testcase: ", testcase.message)
			t.Log("Encoded message: ", encodedMessage)
			t.Error("Encoding failed")
		}

		decodedMessage := Decode(encodedMessage)
		if !bytes.Equal(decodedMessage, testcase.message) {
			t.Log("[FAIL] Xor encoder failed to decode the given message")
			t.Log("Decoded Message: ", decodedMessage)
			t.Log("Testcase: ", testcase.message)
			t.Error("Encoding failed")
		}
	}
	t.Log("[PASS]")
}
