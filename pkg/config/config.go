package config

import (
	"log"
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

func ParseConfig(config []byte) YadroConfig {
	// TODO: Parse the config
	var ParsedConfig YadroConfig
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
