package config

import (
	"log"
	"strconv"

	"gopkg.in/ini.v1"

	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/crypto/symmetric"
	"shlyuz/pkg/encoding/xor"
)

type YadroCrypto struct {
	LpPk        asymmetric.PublicKey
	SymKey      []byte
	XorKey      int
	CompKeypair asymmetric.AsymmetricKeyPair
}

type YadroConfig struct {
	Id            string
	TransportName string
	TskChkTimer   int
	InitSignature []byte
	CryptoConfig  YadroCrypto
}

type LpCrypto struct {
	ImplantPk   asymmetric.PublicKey
	SymKey      []byte
	XorKey      int
	CompKeyPair asymmetric.AsymmetricKeyPair
}

type LpConfig struct {
	Id            string
	TransportName string
	TskChkTimer   int
	InitSignature []byte
	CryptoConfig  LpCrypto
}

func ParseConfig(config []byte) LpConfig {
	var ParsedConfig LpConfig
	cfg, err := ini.Load(config)
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
	ParsedConfig.CryptoConfig.CompKeyPair.PrivKey = (*[32]byte)([]byte(cryptoSec.Key("priv_key").Value()))
	ParsedConfig.CryptoConfig.CompKeyPair.PubKey = *asymmetric.PubFromPriv(ParsedConfig.CryptoConfig.CompKeyPair.PrivKey)
	ParsedConfig.CryptoConfig.ImplantPk = (*[32]byte)([]byte(cryptoSec.Key("imp_pk").Value()))
	ParsedConfig.CryptoConfig.SymKey = []byte(cryptoSec.Key("sym_key").Value())
	xor64Key, err := strconv.ParseInt(cryptoSec.Key("xor_key").Value(), 0, 64)
	if err != nil {
		log.Fatalln("failed to parse config value xkey: ", err)
	}
	ParsedConfig.CryptoConfig.XorKey = int(xor64Key)

	return ParsedConfig
}

func ReadConfig(config []byte, xKey int, symKey []byte) symmetric.SymmetricMessage {
	config = xor.XorMessage(config, xKey)
	rdyConfig := symmetric.Decrypt(config, symKey)
	if rdyConfig.IsEncrypted {
		log.Fatalln("failed to decode config")
	}
	return rdyConfig
}

// Debug function for reading unencrypted configs. DO NOT USE IN PRODUCTION.
func ReadPlaintextConfig(config []byte, symKey []byte) symmetric.SymmetricMessage {
	var rdyConfig symmetric.SymmetricMessage
	rdyConfig.Message = config
	rdyConfig.IsEncrypted = false
	rdyConfig.Key = symKey
	return rdyConfig
}
