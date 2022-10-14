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

func getPath() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("failed to get user home dir: ", err)
	}
	channelPath := userHomeDir + "/tmp/shlyuztest/chan" // TODO: Fixme
	return channelPath
}

func Send(Component *component.Component) error {
	err := func(shlyuzComponent *component.Component) error {
		data := <-shlyuzComponent.CmdChannel
		channelPath := getPath()
		for {
			check_file, err := os.Stat(channelPath)
			if err != nil {
				log.Fatalln("failed to check channel size: ", err)
			}
			if check_file.Size() != 0 {
				continue
			} else {
				break
			}
		}
		err := os.WriteFile(channelPath, data, 0600)
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
	channelPath := getPath()
	// read the contents of the file
	data, err := os.ReadFile(channelPath)
	if err != nil {
		log.Println("failed to read transport channel: ", err)
		return nil, err
	}
	err = os.Truncate(channelPath, 0)
	if err != nil {
		log.Println("WARNING failed to clear transport channel contents: ", err)
		return data, err
	}
	return data, nil
}
