package main

import (
	"log"
	"math/rand"
	"sync"

	// "shlyuz/pkg/crypto/asymmetric"
	"shlyuz/internal/debugLp/pkg/transport"

	"shlyuz/internal/debugLp/pkg/component"
	"shlyuz/internal/debugLp/pkg/config"
	"shlyuz/internal/debugLp/pkg/transaction"
	"shlyuz/pkg/utils/logging"
	"shlyuz/pkg/utils/uname"
)

func generateRandomLPId() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func makeManifest(componentId string) component.ComponentManifest {
	var generatedManifest component.ComponentManifest
	platInfo := uname.GetUname()
	generatedManifest.Lp_hostname = platInfo.Uname.Nodename
	generatedManifest.Lp_id = componentId
	generatedManifest.Lp_os = platInfo.Uname.Sysname
	return generatedManifest
}

func genComponentInfo(lpConfig []byte) component.Component {
	var Component component.Component
	parsedConfig := config.ParseConfig(lpConfig)
	log.SetPrefix(logging.GetLogPrefix())
	log.Println("Started Shlyuz Debug LP")
	Component.CurrentKeypair = parsedConfig.CryptoConfig.CompKeyPair
	Component.InitalKeypair = Component.CurrentKeypair
	Component.Config = parsedConfig
	Component.ComponentId = Component.Config.Id
	Component.XorKey = Component.Config.CryptoConfig.XorKey
	Component.ConfigKey = Component.Config.CryptoConfig.SymKey
	Component.Manifest = makeManifest(Component.ComponentId)
	Component.CurrentImpPubkey = Component.Config.CryptoConfig.ImplantPk

	return Component
}

func main() {
	log.SetPrefix(logging.GetLogPrefix())
	// TODO: make this real
	lpConfig :=
		[]byte(`[lp]
id = ` + generateRandomLPId() + `
transport_name = file_transport
task_check_time = 60
init_signature = b'\xde\xad\xf0\r'
` + `
[crypto]
imp_pk = 4b6d9dgg877lg3231n1gjbn2dgjb79g0gbb77998gggg668866468g334nlb820g
sym_key = BvjTA1o55UmZnuTy
xor_key = 0x6d
priv_key = jnbl37d67g656d617b19l6l02305g68l4d03ngn914800934511b2g13bgdg1021`)
	Component := genComponentInfo(lpConfig)
	transportWg := sync.WaitGroup{}
	defer transportWg.Wait()
	transport, _, err := transport.PrepareTransport(&Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}
	// TODO: This is a loop per implant
	Component.CmdChannel = make(chan []byte)
	initAckInstruction, client, boolSuccess := transaction.RetrieveInitFrame(&Component, transport)
	if !boolSuccess {
		log.Println("implant failed to initalize")
		// TODO: Restart loop here
	}
	transaction.RelayInitFrame(&Component, initAckInstruction, transport)
	// TODO: Register the client here
	log.Println(client)
}
