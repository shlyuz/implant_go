package config

import (
	"math/rand"
	"testing"

	"shlyuz/pkg/crypto/symmetric"
	"shlyuz/pkg/encoding/xor"
)

var plaintextConfigTest = []struct {
	config []byte
}{
	{
		[]byte(`[vzhivlyat]
		id = XDVzqPgKmYYIrCq5uqibXPDfcavJvHVY
		transport_name = transport_bind_tcp_socket
		task_check_time = 60
		init_signature = b'\xde\xad\xf0\r'
		
		[crypto]
		lp_pk = b'\x9d"Q\xf0\xba\xe7\x9b\x92\xffG\xf0I\xb2\xf2\x82\x9cd+\x14\x14\xac\xb5YT\x1f\xd4\xe8lLq \x1a'
		sym_key = BvjTA1o55UmZnuTy
		xor_key = 0x6d
		priv_key = b"\x12\x82\xe0\x89\xdc\xca&\xb6\x02@F'E\x1c\xe89\xcb>\xab}\xd7\xf3\x01\xd8\xf1\x85}8I_\xd2\xa6"`),
	},
}

func TestPlaintextConfig(t *testing.T) {
	t.Parallel()
	blankMsg := symmetric.Encrypt([]byte("ayyylmao"))
	symKey := blankMsg.Key
	for _, testcase := range plaintextConfigTest {
		// "read" the plaintext config
		componentConfig := ReadPlaintextConfig(testcase.config, symKey)
		if componentConfig.Message == nil {
			t.Log("[FAIL] Reading plaintext config failed")
			t.Error("Testcase: ", testcase.config)
		}
		parsedConfig := ParseConfig(componentConfig.Message)
		if parsedConfig.TskChkTimer != 60 {
			t.Log("[FAIL] Plaintext TskChkTimer parsing failed")
			t.Error("Testcase: ", testcase.config)
		}
	}
}

func TestEncryptedConfig(t *testing.T) {
	t.Parallel()
	for _, testcase := range plaintextConfigTest {
		xorKey := rand.Int()
		encryptedConfig := symmetric.Encrypt(testcase.config)
		testConfig := xor.XorMessage(encryptedConfig.Message, xorKey)
		componentConfig := ReadConfig(testConfig, xorKey, encryptedConfig.Key)
		if componentConfig.Message == nil {
			t.Error("[FAIL] Reading encrypted config failed")
		}
		parsedConfig := ParseConfig(componentConfig.Message)
		if parsedConfig.TskChkTimer != 60 {
			t.Error("[FAIL] Encrypted TskChkTimer parsing failed")
		}
	}

}
