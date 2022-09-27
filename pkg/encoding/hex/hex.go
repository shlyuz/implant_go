package hex

import "bytes"

// Encode a given byte array using a 'custom' hex encoder
func Encode(message []byte) []byte {
	message = bytes.Replace(message, []byte("a"), []byte("j"), 0)
	message = bytes.Replace(message, []byte("c"), []byte("n"), 0)
	message = bytes.Replace(message, []byte("e"), []byte("l"), 0)
	message = bytes.Replace(message, []byte("f"), []byte("g"), 0)

	return message
}

// Decode a given byte array using a 'custom' hex encoder
func Decode(message []byte) []byte {
	message = bytes.Replace(message, []byte("j"), []byte("a"), 0)
	message = bytes.Replace(message, []byte("n"), []byte("c"), 0)
	message = bytes.Replace(message, []byte("l"), []byte("e"), 0)
	message = bytes.Replace(message, []byte("g"), []byte("f"), 0)

	return message
}
