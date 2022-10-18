//go:build lp

package lpconfig

import (
	"log"
	"shlyuz/pkg/config"
	"strconv"

	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/encoding/hex"

	"gopkg.in/ini.v1"
)

func ParseConfig(configToParse []byte) config.ShlyuzConfig {
	// TODO: Parse Transport Configuration - #? TransportConfig
	var ParsedConfig config.ShlyuzConfig
	cfg, err := ini.Load(configToParse)
	if err != nil {
		log.Fatalln("failed to load config: ", err)
	}
	lpSec, err := cfg.GetSection("lp")
	if err != nil {
		log.Fatalln("failed to read config section: ", err)
	}

	ParsedConfig.Id = lpSec.Key("id").String()
	ParsedConfig.InitSignature = []byte(lpSec.Key("init_signature").String())
	ParsedConfig.TransportName = lpSec.Key("transport_name").String()
	ParsedConfig.TskChkTimer, err = strconv.Atoi(lpSec.Key("task_check_time").Value())
	if err != nil {
		log.Fatalln("failed to parse config value timer: ", err)
	}

	cryptoSec, err := cfg.GetSection("crypto")
	if err != nil {
		log.Fatalln("failed to read config section 2: ", err)
	}
	compPrivKey := (*[32]byte)(hex.Decode([]byte(cryptoSec.Key("priv_key").Value())))
	ParsedConfig.CryptoConfig.CompKeyPair.PrivKey = (asymmetric.PublicKey)(compPrivKey)
	ParsedConfig.CryptoConfig.CompKeyPair.PubKey = *asymmetric.PubFromPriv(ParsedConfig.CryptoConfig.CompKeyPair.PrivKey)
	impPubKey := hex.Decode([]byte(cryptoSec.Key("imp_pk").Value()))
	ParsedConfig.CryptoConfig.PeerPk = (*[32]byte)(impPubKey)
	ParsedConfig.CryptoConfig.SymKey = []byte(cryptoSec.Key("sym_key").Value())
	xor64Key, err := strconv.ParseInt(cryptoSec.Key("xor_key").Value(), 0, 64)
	if err != nil {
		log.Fatalln("failed to parse config value xkey: ", err)
	}
	ParsedConfig.CryptoConfig.XorKey = int(xor64Key)

	return ParsedConfig
}
