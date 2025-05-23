//go:build linux,implant && (!lp || !teamserver)

package fireinteract

import (
	"bytes"
	"io"
	"shlyuz/pkg/component"
	"strings"
	"sync"
	"testing"
	"time"
	// "os/exec" // For specific error types if needed
)

// Helper function
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 1), // Not directly used by fireinteract's ExecuteCmd
		StdErr: make(chan string, 5), // For start/wait errors
	}
}

// Custom pipe that can be closed and signal closure
type closablePipe struct {
	*bytes.Buffer
	closed    chan struct{}
	closeOnce sync.Once
}

func newClosablePipe() *closablePipe {
	return &closablePipe{
		Buffer: new(bytes.Buffer),
		closed: make(chan struct{}),
	}
}

func (cp *closablePipe) Close() error {
	cp.closeOnce.Do(func() {
		close(cp.closed)
	})
	return nil // bytes.Buffer doesn't have a real Close error
}

func (cp *closablePipe) IsClosed() <-chan struct{} {
	return cp.closed
}

func TestFireInteract_ExecuteCmd_SuccessfulSession(t *testing.T) {
	channels := newTestExecChannels()
	// 'cat' will echo its stdin to stdout
	cmdDetails := component.Command{Id: "fi_success", Args: "cat", Type: "Shell"}

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
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID")
		return 
	}

	inputText := "hello interact\n" // Explicit newline
	errChan := make(chan error, 1) 

	go func() {
		defer inPipeWriter.Close() 
		_, writeErr := inPipeWriter.Write([]byte(inputText))
		if writeErr != nil {
			errChan <- writeErr
		}
	}()
	
	select {
	case <-outPipe.IsClosed(): 
		outputText := outPipe.String()
		// cat outputs exactly what it receives, including the newline.
		if outputText != inputText { 
			t.Errorf("Expected output '%s', got '%s'", inputText, outputText)
		}
	case <-time.After(1 * time.Second): 
		t.Errorf("Timeout waiting for outPipe to be closed. Output so far: '%s'", outPipe.String())
	}
	
	select {
	case e := <-errChan:
		t.Fatalf("Error in input/output goroutines: %v", e)
	default:
	}

	select {
	case errLine := <-channels.StdErr:
		// For a clean 'cat' session ending with EOF on stdin, no error is expected from Wait()
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no command execution error on StdErr for successful cat, got: %s", errLine)
		} else {
			t.Logf("Received non-critical message on StdErr: %s", errLine) // Log if it's not an exit error
		}
	case <-time.After(100 * time.Millisecond): // Give some time for potential errors
		// Good, no error
	}
}

func TestFireInteract_ExecuteCmd_CommandFails(t *testing.T) {
	channels := newTestExecChannels()
	// Using the command from the prompt's code block
	cmdDetails := component.Command{Id: "fi_fail", Args: "bash -c \"echo -n some_output_before_fail; sleep 0.01; exit 1\"", Type: "Shell"}

	inPipeReader, inPipeWriter := io.Pipe() // Input pipe, though command might not read much before exit
	defer inPipeWriter.Close() // Close writer eventually
	defer inPipeReader.Close() // Close reader eventually

	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe) 
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start: %v", err)
	}
	
	select {
	case <-channels.Pid:
		// Consumed PID
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID")
	}

	foundError := false
	outPipeClosed := false
	timeout := time.After(1 * time.Second) // Increased timeout

	for ! (foundError && outPipeClosed) {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") {
				foundError = true
			}
			t.Logf("StdErr: %s", errLine)
		case <-outPipe.IsClosed():
			outPipeClosed = true
			t.Logf("outPipe closed. Content: '%s'", outPipe.String())
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
		t.Errorf("Expected 'exit status 1' on StdErr, but not found")
	}
	if !outPipeClosed {
		t.Errorf("Expected outPipe to be closed")
	}
	// Check output written before failure
	if !strings.Contains(outPipe.String(), "some_output_before_fail") {
		t.Errorf("Expected output 'some_output_before_fail', got '%s'", outPipe.String())
	}
}


func TestFireInteract_ExecuteCmd_StartFail_NotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fi_notfound", Args: "a_very_unlikely_interact_cmd_to_exist_test_foo", Type: "Shell"}

	inPipeReader, inPipeWriter := io.Pipe()
	defer inPipeWriter.Close()
	defer inPipeReader.Close()
	outPipe := newClosablePipe() 

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err == nil {
		t.Fatalf("Expected an error for command not found, got nil")
	}
	// Check the error returned by ExecuteCmd directly
	if !strings.Contains(err.Error(), "executable file not found") && !strings.Contains(err.Error(), "no such file or directory"){
		t.Errorf("Expected error from ExecuteCmd to indicate command not found, got: %v", err)
	}

	// Check the error sent via StdErr channel
	select {
	case errLine := <-channels.StdErr:
		if !strings.Contains(errLine, "executable file not found") && !strings.Contains(errLine, "no such file or directory"){
			t.Errorf("Expected command not found error on StdErr channel, got: %s", errLine)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for error on StdErr channel")
	}

	// PID channel should be empty
	select {
	case pidVal := <-channels.Pid:
		t.Errorf("Expected no PID for command not found, got %d", pidVal)
	case <-time.After(50 * time.Millisecond): // Short timeout, should be quick
		// Good, no PID
	}
}
