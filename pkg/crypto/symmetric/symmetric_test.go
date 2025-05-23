package symmetric

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptSuccess(t *testing.T) {
	plaintexts := [][]byte{
		[]byte(""),
		[]byte("test"),
		[]byte("16bytestring----"), // 16 bytes
		[]byte("This is a longer test message for RC6-CBC."),
	}

	for _, pt :=  range plaintexts {
		encryptedMsg := Encrypt(pt)
		if !encryptedMsg.IsEncrypted {
			t.Errorf("Encrypt(%q) did not set IsEncrypted to true", pt)
			continue
		}
		if len(encryptedMsg.Key) != 32 {
			t.Errorf("Encrypt(%q) did not generate a 32-byte key, got %d bytes", pt, len(encryptedMsg.Key))
			continue
		}

		decryptedMsg := Decrypt(encryptedMsg.Message, encryptedMsg.Key)
		if decryptedMsg.IsEncrypted {
			t.Errorf("Decrypt(Encrypt(%q)) failed, IsEncrypted is true", pt)
			continue
		}
		if !bytes.Equal(decryptedMsg.Message, pt) {
			t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", pt, decryptedMsg.Message, pt)
		}
	}
}

func TestHMACIntegrityTamperedIV(t *testing.T) {
	plaintext := []byte("test message for HMAC integrity")
	encryptedMsg := Encrypt(plaintext)
	if !encryptedMsg.IsEncrypted {
		t.Fatalf("Initial encryption failed")
	}

	// Tamper with IV (first 16 bytes)
	tamperedMessage := make([]byte, len(encryptedMsg.Message))
	copy(tamperedMessage, encryptedMsg.Message)
	if len(tamperedMessage) >= 16 { // Ensure message is long enough for IV
		tamperedMessage[0] ^= 0x01 // Flip the first bit of the IV
	} else {
		t.Fatalf("Encrypted message is too short (%d bytes) to tamper with IV", len(tamperedMessage))
	}


	decryptedMsg := Decrypt(tamperedMessage, encryptedMsg.Key)
	if !decryptedMsg.IsEncrypted {
		t.Errorf("Decryption succeeded with tampered IV, but should have failed.")
	}
}

func TestHMACIntegrityTamperedCiphertext(t *testing.T) {
	plaintext := []byte("test message for HMAC integrity")
	encryptedMsg := Encrypt(plaintext)
	if !encryptedMsg.IsEncrypted {
		t.Fatalf("Initial encryption failed")
	}

	// Tamper with Ciphertext (bytes between IV and HMAC)
	// IV is 16 bytes, HMAC is 32 bytes
	if len(encryptedMsg.Message) < 16+1+32 { // Ensure there's at least 1 byte of ciphertext
		t.Fatalf("Encrypted message is too short to contain ciphertext for tampering: len %d. Needs to be at least %d", len(encryptedMsg.Message), 16+1+32)
	}
	tamperedMessage := make([]byte, len(encryptedMsg.Message))
	copy(tamperedMessage, encryptedMsg.Message)
	tamperedMessage[16] ^= 0x01 // Flip the first bit of the ciphertext part

	decryptedMsg := Decrypt(tamperedMessage, encryptedMsg.Key)
	if !decryptedMsg.IsEncrypted {
		t.Errorf("Decryption succeeded with tampered ciphertext, but should have failed.")
	}
}

func TestHMACIntegrityTamperedHMAC(t *testing.T) {
	plaintext := []byte("test message for HMAC integrity")
	encryptedMsg := Encrypt(plaintext)
	if !encryptedMsg.IsEncrypted {
		t.Fatalf("Initial encryption failed")
	}

	// Tamper with HMAC (last 32 bytes)
	if len(encryptedMsg.Message) < 32 {
		t.Fatalf("Encrypted message is too short (%d bytes) to tamper with HMAC", len(encryptedMsg.Message))
	}
	tamperedMessage := make([]byte, len(encryptedMsg.Message))
	copy(tamperedMessage, encryptedMsg.Message)
	tamperedMessage[len(tamperedMessage)-1] ^= 0x01 // Flip the last bit of the HMAC

	decryptedMsg := Decrypt(tamperedMessage, encryptedMsg.Key)
	if !decryptedMsg.IsEncrypted {
		t.Errorf("Decryption succeeded with tampered HMAC, but should have failed.")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	plaintext := []byte("test message for wrong key")
	encryptedMsg := Encrypt(plaintext)
	if !encryptedMsg.IsEncrypted {
		t.Fatalf("Initial encryption failed")
	}

	wrongKey := generateKey() // Generate a different key
	// Ensure keys are actually different, highly improbable they are the same
	if bytes.Equal(encryptedMsg.Key, wrongKey) {
		t.Log("Warning: Generated wrong key is identical to original key. Retrying.")
		wrongKey = generateKey()
		if bytes.Equal(encryptedMsg.Key, wrongKey) {
			t.Fatalf("Failed to generate a different key for TestDecryptWrongKey")
		}
	}


	decryptedMsg := Decrypt(encryptedMsg.Message, wrongKey)
	if !decryptedMsg.IsEncrypted {
		t.Errorf("Decryption succeeded with wrong key, but should have failed.")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := generateKey()
	// Minimum valid length: IV (16) + 1 block Ciphertext (16) + HMAC (32) = 64 bytes
	shortMessages := [][]byte{
		make([]byte, 0),    // Empty
		make([]byte, 10),   // Less than IV
		make([]byte, 15),   // Less than IV
		make([]byte, 16),   // IV only, no ciphertext, no HMAC
		make([]byte, 32),   // IV + 16B (could be Ciphertext or HMAC part, but not both + missing one)
		make([]byte, 47),   // IV + HMAC (no ciphertext) OR IV + 16B CT + partial HMAC
		make([]byte, 48),   // IV + HMAC (no ciphertext) OR IV + 16B CT + partial HMAC
		make([]byte, 63),   // Just under minimum valid (IV+CipherBlock+HMAC = 16+16+32=64)
	}

	for i, msg := range shortMessages {
		decryptedMsg := Decrypt(msg, key)
		if !decryptedMsg.IsEncrypted {
			t.Errorf("Test %d: Decryption succeeded with too short message (len %d), but should have failed.", i, len(msg))
		}
	}
}
