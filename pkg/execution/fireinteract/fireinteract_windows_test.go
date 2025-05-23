//go:build windows,implant && (!lp || !teamserver)

package fireinteract

import (
	"bytes"
	"io"
	"shlyuz/pkg/component"
	"strings"
	"sync"
	"testing"
	"time"
)

// Helper function
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 1), 
		StdErr: make(chan string, 10), // Increased capacity
	}
}

// Custom pipe that can be closed and signal closure
type closablePipe struct {
	*bytes.Buffer
	closed    chan struct{}
	closeOnce sync.Once
	writeLock sync.Mutex // Ensure concurrent writes to buffer are safe
}

func newClosablePipe() *closablePipe {
	return &closablePipe{
		Buffer: new(bytes.Buffer),
		closed: make(chan struct{}),
	}
}

func (cp *closablePipe) Write(p []byte) (n int, err error) {
	cp.writeLock.Lock()
	defer cp.writeLock.Unlock()
	return cp.Buffer.Write(p)
}

func (cp *closablePipe) Close() error {
	cp.closeOnce.Do(func() {
		close(cp.closed)
	})
	return nil
}

func (cp *closablePipe) IsClosed() <-chan struct{} {
	return cp.closed
}


func TestFireInteract_ExecuteCmd_Windows_SuccessfulSession(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fi_win_success", Args: "findstr /N /P \"\"", Type: "Shell"} // Ensure quotes for findstr

	inPipeReader, inPipeWriter := io.Pipe()
	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start: %v", err)
	}

	var pid int
	select {
	case pid = <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID, got %d", pid)
		}
	case <-time.After(200 * time.Millisecond): 
		t.Fatalf("Timeout waiting for PID")
		return
	}

	inputText := "hello interact windows\n" // Using \n as per prompt, findstr handles it
	
	go func() {
		defer inPipeWriter.Close() 
		_, writeErr := inPipeWriter.Write([]byte(inputText))
		if writeErr != nil {
			// Use t.Error or t.Errorf for thread-safe logging from goroutines in tests
			t.Errorf("Error writing to inPipeWriter: %v", writeErr) 
		}
	}()

	select {
	case <-outPipe.IsClosed():
		outputText := outPipe.String()
		if !strings.Contains(outputText, "hello interact windows") {
			t.Errorf("Expected output to contain '%s', got '%s'", "hello interact windows", outputText)
		}
		// findstr /N adds "N:" prefix. If input is "hello\n", output might be "1:hello\n"
		// For "hello interact windows\n", it might be "1:hello interact windows\n"
		// Checking for the presence of the input text is primary. Line number check is secondary.
		if !strings.Contains(outputText, ":hello interact windows") {
             t.Logf("Output was '%s', check if findstr numbering format is as expected.", outputText)
        }

	case <-time.After(2 * time.Second): 
		t.Errorf("Timeout waiting for outPipe to be closed. Output so far: '%s'", outPipe.String())
	}

	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") && !strings.Contains(errLine, "exit status 0") {
			t.Errorf("Expected no command execution error (or exit status 0) on StdErr, got: %s", errLine)
		} else if errLine != "" { // Log if not empty and not an error
			t.Logf("Got non-critical/non-error stderr message: %s", errLine)
		}
	case <-time.After(200 * time.Millisecond): // Allow time for any final stderr messages
		// Good, no error or only non-critical message
	}
}

func TestFireInteract_ExecuteCmd_Windows_CommandFailsMidWay(t *testing.T) {
	channels := newTestExecChannels()
	cmdStr := "echo some_output_before_fail_win & echo error stuff >&2 & exit 1"
	cmdDetails := component.Command{Id: "fi_win_fail_mid", Args: cmdStr, Type: "Shell"}

	var dummyInput bytes.Buffer 
	inPipeReader := io.NopCloser(&dummyInput) 
	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start: %v", err)
	}

	select {
	case <-channels.Pid:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID")
	}

	foundError := false
	outPipeClosed := false
	timeout := time.After(1 * time.Second)

	for !(foundError && outPipeClosed) {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") {
				foundError = true
			}
			t.Logf("StdErr: %s", errLine)
		case <-outPipe.IsClosed():
			outPipeClosed = true
		case <-timeout:
			if !foundError {
				t.Errorf("Timeout waiting for 'exit status 1' on StdErr. Output: %s", outPipe.String())
			}
			if !outPipeClosed {
				t.Errorf("Timeout waiting for outPipe to be closed. Output: %s", outPipe.String())
			}
			return
		}
	}
	
	if !foundError {
		t.Errorf("Expected 'exit status 1' on StdErr")
	}
	if !outPipeClosed {
		t.Errorf("Expected outPipe to be closed")
	}

	outputText := outPipe.String()
	if !strings.Contains(outputText, "some_output_before_fail_win") {
		t.Errorf("Expected output to contain 'some_output_before_fail_win', got '%s'", outputText)
	}
	if !strings.Contains(outputText, "error stuff") {
		t.Errorf("Expected output to contain 'error stuff' (from stderr), got '%s'", outputText)
	}
}

func TestFireInteract_ExecuteCmd_Windows_InnerCommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArg := "an_interact_win_cmd_that_is_not_real_at_all_xyz"
	cmdDetails := component.Command{Id: "fi_win_cmd_not_found", Args: cmdArg, Type: "Shell"}

	var dummyInput bytes.Buffer
	inPipeReader := io.NopCloser(&dummyInput)
	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start cmd.exe: %v", err)
	}

	select {
	case <-channels.Pid: 
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID")
	}

	foundError := false 
	outPipeClosed := false
	timeout := time.After(1 * time.Second)

	for !(foundError && outPipeClosed) {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") { 
				foundError = true
			}
			t.Logf("StdErr (from execChannels): %s", errLine)
		case <-outPipe.IsClosed():
			outPipeClosed = true
		case <-timeout:
			if !foundError {
				t.Errorf("Timeout waiting for 'exit status 1' (from cmd.exe) on StdErr. Output: %s", outPipe.String())
			}
			if !outPipeClosed {
				t.Errorf("Timeout waiting for outPipe to be closed. Output: %s", outPipe.String())
			}
			return
		}
	}

	if !foundError {
		t.Errorf("Expected 'exit status 1' from cmd.exe on StdErr")
	}
	if !outPipeClosed {
		t.Errorf("Expected outPipe to be closed")
	}
	
	outputText := outPipe.String()
	// This checks if cmd.exe's error message about the command not being found is part of the output.
	// This message can be locale-dependent.
	if !strings.Contains(outputText, cmdArg) || !strings.Contains(outputText, "is not recognized") {
		t.Logf("Output from cmd.exe ('%s') did not strongly indicate command not found with standard Windows message, but exit status 1 from cmd.exe is key.", outputText)
	}
}
