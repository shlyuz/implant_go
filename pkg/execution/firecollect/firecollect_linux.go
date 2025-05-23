//go:build linux,implant && (!lp || !teamserver)

package firecollect

import (
	"bytes"
	"errors" // For returning new errors
	"fmt"    // For fmt.Errorf
	"io"
	"log"
	"os"
	"os/exec"
	"shlyuz/pkg/component"
	"strings" // For strings.Split
	"sync"    // For coordinating reader goroutine
)

// Execute runs the command specified by cmdDetails and collects its standard output in-memory.
// Standard error is captured and returned as part of an error if the command fails.
func Execute(cmdDetails component.Command, execChannels *component.ComponentExecutionChannel) (string, error) {
	log.Printf("CmdID %s: Preparing to execute and collect output for: %s", cmdDetails.Id, cmdDetails.Args)

	cmdArgs := strings.Split(cmdDetails.Args, " ")
	command := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Create an in-memory pipe for stdout
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("CmdID %s: Error creating stdout pipe: %v", cmdDetails.Id, err)
		return "", fmt.Errorf("failed to create stdout pipe for cmd %s: %w", cmdDetails.Id, err)
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
		defer r.Close() // Close reader end when done
		log.Printf("CmdID %s: Starting to read from stdout pipe.", cmdDetails.Id)
		if _, copyErr := io.Copy(&outputBuf, r); copyErr != nil {
			// This error can happen if the pipe is closed unexpectedly.
			// It's often not critical if the command itself manages its lifecycle.
			log.Printf("CmdID %s: Error copying stdout from pipe: %v", cmdDetails.Id, copyErr)
		}
		log.Printf("CmdID %s: Finished reading from stdout pipe.", cmdDetails.Id)
	}()

	// Start the command
	if err := command.Start(); err != nil {
		log.Printf("CmdID %s: Error starting command: %v", cmdDetails.Id, err)
		_ = w.Close() // Close writer end on error (best effort)
		_ = r.Close() // Close reader end as well (best effort), goroutine might be stuck otherwise
		// wg.Wait() // Not strictly necessary to wait if Start fails, but ensures goroutine exits if it somehow progressed
		return "", fmt.Errorf("failed to start command %s: %w. Stderr: %s", cmdDetails.Id, err, stderrBuf.String())
	}
	log.Printf("CmdID %s: Command started successfully (PID: %d).", cmdDetails.Id, command.Process.Pid)
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default:
			log.Printf("CmdID %s: PID channel full, dropping PID for command %s", cmdDetails.Id, cmdDetails.Args)
		}
	}

	// Close the writer end of the pipe. The command has its own copy of the fd.
	// This is crucial so that the reader goroutine sees EOF when the command finishes writing.
	if err := w.Close(); err != nil {
		log.Printf("CmdID %s: Error closing command's stdout pipe writer: %v", cmdDetails.Id, err)
		// This is not necessarily fatal for command execution but might affect output collection.
		// For now, just log. The command might fail later if it depends on writing to stdout.
	}

	// Wait for the command to finish
	waitErr := command.Wait()

	// Wait for the reader goroutine to finish processing all output.
	// This must happen AFTER command.Wait() to ensure all output is flushed
	// from the command's internal buffers and the pipe is broken from the writer side.
	wg.Wait()

	if waitErr != nil {
		log.Printf("CmdID %s: Command finished with error: %v. Stderr: %s", cmdDetails.Id, waitErr, stderrBuf.String())
		// Include stderr in the error message if available
		errMsg := fmt.Sprintf("command %s finished with error: %v", cmdDetails.Id, waitErr)
		if stderrBuf.Len() > 0 {
			errMsg = fmt.Sprintf("%s. Stderr: %s", errMsg, stderrBuf.String())
		}
		return outputBuf.String(), errors.New(errMsg) // Return collected output along with error
	}

	log.Printf("CmdID %s: Command finished successfully. Collected %d bytes of output.", cmdDetails.Id, outputBuf.Len())
	return outputBuf.String(), nil
}
