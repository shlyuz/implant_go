//go:build testing

package config

import (
	"log"
	"math/rand"
	"strconv"
	"testing"

	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/crypto/symmetric"
	"shlyuz/pkg/encoding/hex"
	"shlyuz/pkg/encoding/xor"

	"gopkg.in/ini.v1"
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

func ParseConfig(configToParse []byte) ShlyuzConfig {
	// TODO: Parse Transport Configuration - #? TransportConfig
	var ParsedConfig ShlyuzConfig
	cfg, err := ini.Load(configToParse)
	if err != nil {
		log.Fatalln("failed to load config: ", err)
	}
	vzhivlyatSec, err := cfg.GetSection("vzhivlyat")
	if err != nil {
		log.Fatalln("failed to read config section: ", err)
	}

	ParsedConfig.Id = vzhivlyatSec.Key("id").String()
	ParsedConfig.InitSignature = []byte(vzhivlyatSec.Key("init_signature").String())
	ParsedConfig.TransportName = vzhivlyatSec.Key("transport_name").String()
	ParsedConfig.TskChkTimer, err = strconv.Atoi(vzhivlyatSec.Key("task_check_time").Value())
	if err != nil {
		log.Fatalln("failed to parse config value timer: ", err)
	}

	cryptoSec, err := cfg.GetSection("crypto")
	if err != nil {
		log.Fatalln("failed to read config section 2: ", err)
	}
	compPrivKey := (*[32]byte)(hex.Decode([]byte(cryptoSec.Key("priv_key").Value())))
	ParsedConfig.CryptoConfig.CompKeyPair.PrivKey = (*[32]byte)(compPrivKey)
	ParsedConfig.CryptoConfig.CompKeyPair.PubKey = *asymmetric.PubFromPriv(ParsedConfig.CryptoConfig.CompKeyPair.PrivKey)
	lpPubKey := (*[32]byte)(hex.Decode([]byte(cryptoSec.Key("lp_pk").Value())))
	ParsedConfig.CryptoConfig.PeerPk = (*[32]byte)(lpPubKey)
	ParsedConfig.CryptoConfig.SymKey = []byte(cryptoSec.Key("sym_key").Value())
	xor64Key, err := strconv.ParseInt(cryptoSec.Key("xor_key").Value(), 0, 64)
	if err != nil {
		log.Fatalln("failed to parse config value xkey: ", err)
	}
	ParsedConfig.CryptoConfig.XorKey = int(xor64Key)

	return ParsedConfig
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
