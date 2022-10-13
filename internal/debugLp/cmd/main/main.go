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
	lpConfig :=
		[]byte(`[lp]
id = ` + generateRandomLPId() + `
transport_name = file_transport
task_check_time = 60
init_signature = b'\xde\xad\xf0\r'
` + `
[crypto]
imp_pk = b'\xd2Q` + "`" + `DǓ0\xa8\n\xde\x11~n0|\n\x85\xf3\xcb\xee\x19\xaf\v\x93s\x80WO\xb5\xdegH'` + `
sym_key = BvjTA1o55UmZnuTy
xor_key = 0x6d
priv_key = "b'a\\xb2H\\x9a\\xb1\\xcb\\xa5\\xf30\\x1"`)
	log.Println(string(lpConfig))
	Component := genComponentInfo(lpConfig)
	transportWg := sync.WaitGroup{}
	defer transportWg.Wait()
	transport, _, err := transport.PrepareTransport(&Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}
	Component.CmdChannel = make(chan []byte)
	transaction.RetrieveInitFrame(&Component, transport)

	// TODO: Receive actual manifest
}
