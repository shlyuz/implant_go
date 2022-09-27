package xor

func XorMessage(message []byte, key int) []byte {
	encodedMessage := make([]byte, len(message))
	copy(encodedMessage, message)
	for i := 0; i < len(encodedMessage); i += 1 {
		encodedMessage[i] = encodedMessage[i] ^ byte(key)
	}
	return encodedMessage
}
