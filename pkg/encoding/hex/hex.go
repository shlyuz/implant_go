package hex

import (
	"bytes"
	"encoding/hex"
)

func replace(message []byte, target string, replacement string) []byte {
	// targetInt, _ := strconv.Atoi(target)
	// replacementInt, _ := strconv.Atoi(replacement)
	message = bytes.Replace(message, []byte(target), []byte(replacement), -1)
	return message
}

// Encode a given byte array using a 'custom' hex encoder
func Encode(message []byte) []byte {
	encodedMessage := make([]byte, hex.EncodedLen(len(message)))
	hex.Encode(encodedMessage, message)
	encodedMessage = replace(encodedMessage, "a", "j")
	encodedMessage = replace(encodedMessage, "c", "n")
	encodedMessage = replace(encodedMessage, "e", "l")
	encodedMessage = replace(encodedMessage, "f", "g")

	return encodedMessage
}

// Decode a given byte array using a 'custom' hex encoder
func Decode(message []byte) []byte {
	finalMessage := make([]byte, hex.DecodedLen(len(message)))
	decodedMessage := message
	decodedMessage = replace(decodedMessage, "g", "f")
	decodedMessage = replace(decodedMessage, "l", "e")
	decodedMessage = replace(decodedMessage, "n", "c")
	decodedMessage = replace(decodedMessage, "j", "a")
	hex.Decode(finalMessage, decodedMessage)

	return finalMessage
}
