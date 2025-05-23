//go:build darwin,implant && (!lp || !teamserver)

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
		StdErr: make(chan string, 10), 
	}
}

// Custom pipe that can be closed and signal closure
type closablePipe struct {
	*bytes.Buffer
	closed    chan struct{}
	closeOnce sync.Once
	writeLock sync.Mutex
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


func TestFireInteract_ExecuteCmd_Darwin_SuccessfulSession(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fi_darwin_success", Args: "cat", Type: "Shell"}

	inPipeReader, inPipeWriter := io.Pipe()
	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start (Darwin): %v", err)
	}

	var pid int
	select {
	case pid = <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Darwin), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID (Darwin)")
		return
	}

	inputText := "hello interact darwin\n" // Explicit newline
	errChan := make(chan error, 1)

	go func() {
		defer inPipeWriter.Close()
		_, writeErr := inPipeWriter.Write([]byte(inputText))
		if writeErr != nil {
			errChan <- writeErr // Send error to channel for main test goroutine to check
		}
	}()
	
	select {
	case <-outPipe.IsClosed():
		outputText := outPipe.String()
		// On Darwin, cat should output exactly what it receives.
		if outputText != inputText { 
			t.Errorf("Expected output '%s' (Darwin), got '%s'", inputText, outputText)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for outPipe to be closed (Darwin). Output so far: '%s'", outPipe.String())
	}
	
	select {
	case e := <-errChan:
		t.Fatalf("Error in input/output goroutines (Darwin): %v", e)
	default:
	}

	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") && !strings.Contains(errLine, "exit status 0") {
			t.Errorf("Expected no command execution error (or exit status 0) on StdErr (Darwin), got: %s", errLine)
		} else {
			t.Logf("Got non-critical stderr message (Darwin Success): %s", errLine)
		}
	case <-time.After(100 * time.Millisecond): // Wait a bit for any potential async error.
		// Good, no error.
	}
}

func TestFireInteract_ExecuteCmd_Darwin_CommandFailsMidWay(t *testing.T) {
	channels := newTestExecChannels()
	// Corrected command string quoting
	cmdStr := "bash -c \"echo -n some_darwin_output; echo 'darwin error stuff' >&2; exit 1\""
	cmdDetails := component.Command{Id: "fi_darwin_fail_mid", Args: cmdStr, Type: "Shell"}

	var dummyInput bytes.Buffer // Command doesn't read extensive input before failing
	inPipeReader := io.NopCloser(&dummyInput) 
	outPipe := newClosablePipe()

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err != nil {
		t.Fatalf("ExecuteCmd failed to start (Darwin): %v", err)
	}

	select {
	case <-channels.Pid:
		// PID consumed
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for PID (Darwin)")
	}

	foundError := false
	outPipeClosed := false
	timeout := time.After(1 * time.Second) // Allow time for command to run and pipes to close

	for !(foundError && outPipeClosed) {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") {
				foundError = true
			}
			t.Logf("StdErr (Darwin FailMidWay): %s", errLine)
		case <-outPipe.IsClosed():
			outPipeClosed = true
			t.Logf("outPipe closed (Darwin FailMidWay). Content: '%s'", outPipe.String())
		case <-timeout:
			if !foundError {
				t.Errorf("Timeout waiting for 'exit status 1' on StdErr (Darwin). Output: %s", outPipe.String())
			}
			if !outPipeClosed {
				t.Errorf("Timeout waiting for outPipe to be closed (Darwin). Output: %s", outPipe.String())
			}
			return
		}
	}
	
	if !foundError {
		t.Errorf("Expected 'exit status 1' on StdErr (Darwin)")
	}
	if !outPipeClosed {
		t.Errorf("Expected outPipe to be closed (Darwin)")
	}

	outputText := outPipe.String()
	if !strings.Contains(outputText, "some_darwin_output") {
		t.Errorf("Expected output to contain 'some_darwin_output' (Darwin), got '%s'", outputText)
	}
	// Stderr from the command is also written to outPipe
	if !strings.Contains(outputText, "darwin error stuff") {
		t.Errorf("Expected output to contain 'darwin error stuff' (Darwin), got '%s'", outputText)
	}
}


func TestFireInteract_ExecuteCmd_Darwin_StartFail_NotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArg := "an_interact_darwin_cmd_that_is_not_real_at_all_xyz"
	cmdDetails := component.Command{Id: "fi_darwin_notfound", Args: cmdArg, Type: "Shell"}

	var dummyInput bytes.Buffer
	inPipeReader := io.NopCloser(&dummyInput)
	// Not using defer inPipeReader.Close() as it's a NopCloser on a Buffer.
	outPipe := newClosablePipe() 

	err := ExecuteCmd(cmdDetails, channels, inPipeReader, outPipe)
	if err == nil {
		t.Fatalf("Expected an error for command not found (Darwin), got nil")
	}
	// Check the error returned by ExecuteCmd directly
	if !strings.Contains(err.Error(), "executable file not found") && !strings.Contains(err.Error(), "no such file or directory"){
		t.Errorf("Expected error from ExecuteCmd to indicate command not found (Darwin), got: %v", err)
	}

	// Check the error sent via StdErr channel (from the Start() path)
	select {
	case errLine := <-channels.StdErr:
		// The error from Start() is prefixed in fireinteract_darwin.go
		expectedErrSubstring := "Error starting interactive command"
		if !strings.Contains(errLine, expectedErrSubstring) {
			t.Errorf("Expected error on StdErr channel to contain '%s' (Darwin), got: %s", expectedErrSubstring, errLine)
		}
		if !strings.Contains(errLine, "executable file not found") && !strings.Contains(errLine, "no such file or directory"){
			t.Errorf("Expected error on StdErr channel to indicate command not found (Darwin), got: %s", errLine)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for error on StdErr channel (Darwin)")
	}

	// PID channel should be empty
	select {
	case pidVal := <-channels.Pid:
		t.Errorf("Expected no PID for start fail (Darwin), got %d", pidVal)
	default: // Good, no PID
	}
}
