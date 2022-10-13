package filetransport

import (
	"log"
	"os"
	"shlyuz/internal/debugLp/pkg/component"
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
	err := func(shlyuzComponent *component.Component) error {
		data := <-shlyuzComponent.CmdChannel
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
		return err
	}(Component)
	if err != nil {
		return err
	}
	return nil
}

func Recv(Component *component.Component) ([]byte, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("failed to get user home dir: ", err)
	}
	channelPath := userHomeDir + "/tmp/shlyuztest/chan"
	// read the contents of the file
	data, err := os.ReadFile(channelPath)
	if err != nil {
		log.Println("failed to read transport channel: ", err)
		return nil, err
	}
	// err = os.Truncate(channelPath, 0)
	// if err != nil {
	// 	log.Println("WARNING failed to clear transport channel contents: ", err)
	// 	return data, err
	// }
	return data, nil
}
