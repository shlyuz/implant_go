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
	func(shlyuzComponent *component.Component) []byte {
		var data []byte
		data = <-shlyuzComponent.CmdChannel
		err := os.WriteFile("~/tmp/shlyuztest/chan", data, 0600)
		if err != nil {
			log.Println("something went wrong")
		}
		close(shlyuzComponent.CmdChannel)
		return data
	}(Component)

	return nil
}
