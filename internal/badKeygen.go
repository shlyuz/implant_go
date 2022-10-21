package main

import (
	"fmt"
	"log"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/encoding/hex"
	"shlyuz/pkg/utils/idgen"
)

func main() {
	lpKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Fatalln("failed to generate keypair: ", err)
	}
	impKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Fatalln("failed to generate keypair: ", err)
	}
	encodedLpPubKey := lpKeyPair.PubKey[:]
	encodedLpPubKey = hex.Encode(encodedLpPubKey)
	encodedLpPrivKey := lpKeyPair.PrivKey[:]
	encodedLpPrivKey = hex.Encode(encodedLpPrivKey)
	encodedImpPubKey := impKeyPair.PubKey[:]
	encodedImpPubKey = hex.Encode(encodedImpPubKey)
	encodedImpPrivKey := impKeyPair.PrivKey[:]
	encodedImpPrivKey = hex.Encode(encodedImpPrivKey)
	fmt.Println("LP Pub: ", string(encodedLpPubKey))
	fmt.Println("LP Priv: ", string(encodedLpPrivKey))
	fmt.Println("Imp Pub: ", string(encodedImpPubKey))
	fmt.Println("Imp Priv: ", string(encodedImpPrivKey))
	fmt.Println(idgen.GenerateTxId())
}
