package hex

import "bytes"

// Encode a given byte array using a 'custom' hex encoder
func Encode(message []byte) []byte {
	message = bytes.Replace(message, []byte("a"), []byte("j"), -1)
	message = bytes.Replace(message, []byte("c"), []byte("n"), -1)
	message = bytes.Replace(message, []byte("e"), []byte("l"), -1)
	message = bytes.Replace(message, []byte("f"), []byte("g"), -1)

	return message
}

// Decode a given byte array using a 'custom' hex encoder
func Decode(message []byte) []byte {
	message = bytes.Replace(message, []byte("j"), []byte("a"), -1)
	message = bytes.Replace(message, []byte("n"), []byte("c"), -1)
	message = bytes.Replace(message, []byte("l"), []byte("e"), -1)
	message = bytes.Replace(message, []byte("g"), []byte("f"), -1)

	return message
}
