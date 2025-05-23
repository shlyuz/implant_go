//go:build linux,implant && (!lp || !teamserver)

package fireforget

import (
	"bytes" // For inPipe
	"io"    // For io.NopCloser
	"shlyuz/pkg/component"
	"strings"
	"testing"
	"time"
	// "os/exec" // For specific error types if needed
)

// Helper function
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 1), // Should not be used by fireforget
		StdErr: make(chan string, 5), // For start errors
	}
}

func TestFireForget_ExecuteCmd_Success(t *testing.T) {
	channels := newTestExecChannels()
	// sleep 0.1 ensures the command runs for a bit but test doesn't hang on it.
	cmdDetails := component.Command{Id: "ff_success", Args: "sleep 0.1", Type: "Shell"}

	// Use a channel to signal ExecuteCmd completion
	done := make(chan error)
	go func() {
		done <- ExecuteCmd(cmdDetails, channels, nil)
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("ExecuteCmd failed: %v", err)
		}
	case <-time.After(100 * time.Millisecond): // Should return much faster
		t.Fatalf("ExecuteCmd did not return quickly, possibly waiting for command.")
	}

	// Check PID
	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID, got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID")
	}

	// StdErr should be empty for successful start
	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start, got: %s", errLine)
	default:
		// Good
	}
}

func TestFireForget_ExecuteCmd_StartFail_NotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArgs := "a_fireforget_command_that_does_not_exist_foobar" // Using command from prompt
	cmdDetails := component.Command{Id: "ff_notfound", Args: cmdArgs, Type: "Shell"}

	err := ExecuteCmd(cmdDetails, channels, nil)

	if err == nil {
		t.Fatalf("Expected an error for command not found, got nil")
	}
	if !strings.Contains(err.Error(), "executable file not found") && !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Expected error to indicate command not found, got: %v", err)
	}

	// Check StdErr for the error
	select {
	case errLine := <-channels.StdErr:
		if !strings.Contains(errLine, "executable file not found") && !strings.Contains(errLine, "no such file or directory") {
			t.Errorf("Expected command not found error on StdErr, got: %s", errLine)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for error on StdErr")
	}

	// PID channel should be empty
	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID to be sent on command start failure, got %d", pid)
	default:
		// Good
	}
}

func TestFireForget_ExecuteCmd_WithInput(t *testing.T) {
	channels := newTestExecChannels()
	// Using "cat" which will read its stdin. The actual output of cat is logged by fireforget, not tested here.
	// We are testing if providing input blocks or causes ExecuteCmd to fail.
	cmdDetails := component.Command{Id: "ff_with_input", Args: "cat", Type: "Shell"}
	
	inputData := "hello fireforget input"
	inPipe := io.NopCloser(bytes.NewBufferString(inputData)) // Create a reader for the input

	done := make(chan error)
	go func() {
		done <- ExecuteCmd(cmdDetails, channels, inPipe)
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("ExecuteCmd failed with input: %v", err)
		}
	case <-time.After(100 * time.Millisecond): // Should return quickly
		t.Fatalf("ExecuteCmd did not return quickly with input, possibly waiting.")
	}

	// Check PID
	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID with input, got %d", pid)
		}
		// It's good practice to try and kill the 'cat' process here if possible,
		// as it will be running in the background. However, test framework might not allow direct process kill easily.
		// For now, assume OS or environment will clean it up.
		// On Linux, you could try: exec.Command("kill", fmt.Sprintf("%d", pid)).Run()
		// But this adds complexity and flakiness to the test.
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID with input")
	}

	// StdErr should be empty for successful start
	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start with input, got: %s", errLine)
	default:
		// Good
	}
}
