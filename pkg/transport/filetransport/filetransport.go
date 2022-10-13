package filetransport

import (
	"log"
	"os"
	"shlyuz/pkg/component"
)

type TransportInfo struct {
	Name        string
	Author      string
	Description string
	Version     string
}

type Transport interface {
	Info() TransportInfo
}

func Send(Component *component.Component) error {
	data := func(shlyuzComponent *component.Component) []byte {
		var data []byte
		data = <-shlyuzComponent.CmdChannel
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalln("failed to get user home dir: ", err)
		}
		channelPath := userHomeDir + "/tmp/shlyuztest/chan"
		err = os.WriteFile(channelPath, data, 0600)
		if err != nil {
			log.Println("something went wrong: ", err)
		}
		close(shlyuzComponent.CmdChannel)
		return data
	}(Component)
	log.Println(data)
	return nil
}
