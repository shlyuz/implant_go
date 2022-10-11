package ipc

// This is basically https://github.com/davidelorenzoli/named-pipe-ipc/

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// Create a named pipe
func CreateNamedPipe() string {
	tmpDir, _ := ioutil.TempDir("", "shlyuz") // TODO: randomize me somehow
	namedPipe := filepath.Join(tmpDir, "stdout")

	if err := syscall.Mkfifo(namedPipe, 0600); err != nil {
		log.Printf("failed to create named pipe %s. Error: %s\n", tmpDir, err.Error())
	} else {
		log.Printf("Created named pipe %s", tmpDir)
	}

	return namedPipe
}

// Open a named pipe for reading
func Read(namedPipe string) string {
	stdout, _ := os.OpenFile(namedPipe, os.O_RDONLY, 0600)
	var buff bytes.Buffer

	if _, err := io.Copy(&buff, stdout); err != nil {
		log.Printf("failed to read pipe. Error: %s", err)
	}

	if err := stdout.Close(); err != nil {
		log.Printf("failed to close stream. Error: %s", err)
	}

	return buff.String()
}

func Write(namedPipe string, content []byte) error {
	stdout, _ := os.OpenFile(namedPipe, os.O_RDWR, 0600)
	if _, err := stdout.Write(content); err != nil {
		log.Printf("error writing bytes %s", err.Error())
		return err
	}
	if err := stdout.Close(); err != nil {
		log.Printf("error closing writer: %s", err.Error())
		return err
	}
	return nil
}
