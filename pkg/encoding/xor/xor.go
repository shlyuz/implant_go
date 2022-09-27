package xor

func XorMessage(message []byte, key int) []byte {
	for i := 0; i < len(message); i += 1 {
		message[i] = message[i] ^ byte(key)
	}
	return message
}
