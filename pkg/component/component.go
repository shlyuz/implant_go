package component

import (
	"shlyuz/pkg/config"
	"shlyuz/pkg/crypto/asymmetric"
)

type Component struct {
	// logger             log.Logger
	ConfigFile         string
	ConfigKey          []byte
	Config             config.YadroConfig
	Manifest           ComponentManifest
	InitalKeypair      asymmetric.AsymmetricKeyPair
	CurrentKeypair     asymmetric.AsymmetricKeyPair
	XorKey             int
	CmdQueue           []byte
	CmdProcessingQueue []byte
	CmdDoneQueue       []byte
}

type ComponentManifest struct {
	Implant_id       string
	Implant_os       string
	Implant_hostname string
}
