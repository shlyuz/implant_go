package config

import (
	"log"

	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/crypto/symmetric"
	"shlyuz/pkg/encoding/xor"
)

type ShlyuzCrypto struct {
	PeerPk      asymmetric.PublicKey
	SymKey      []byte
	XorKey      int
	CompKeyPair asymmetric.AsymmetricKeyPair
}

type ShlyuzConfig struct {
	Id            string
	TransportName string
	TskChkTimer   int
	InitSignature []byte
	CryptoConfig  ShlyuzCrypto
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
