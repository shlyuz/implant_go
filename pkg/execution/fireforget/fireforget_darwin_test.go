//go:build darwin,implant && (!lp || !teamserver)

package fireforget

import (
	"bytes" // For inPipe
	"io"    // For io.NopCloser
	"shlyuz/pkg/component"
	"strings"
	"testing"
	"time"
)

// Helper function
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 1), // Not used
		StdErr: make(chan string, 5), // For start errors from ExecuteCmd itself
	}
}

func TestFireForget_ExecuteCmd_Darwin_Success(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "ff_darwin_success", Args: "sleep 0.1", Type: "Shell"}

	done := make(chan error)
	go func() {
		done <- ExecuteCmd(cmdDetails, channels, nil)
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("ExecuteCmd failed (Darwin): %v", err)
		}
	case <-time.After(100 * time.Millisecond): 
		t.Fatalf("ExecuteCmd did not return quickly (Darwin), possibly waiting.")
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Darwin), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Darwin)")
	}

	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start (Darwin), got: %s", errLine)
	default:
	}
}

func TestFireForget_ExecuteCmd_Darwin_StartFail_NotFound(t *testing.T) {
	channels := newTestExecChannels()
	// Using the command from the prompt's code block
	cmdArgs := "a_fireforget_cmd_that_should_not_exist_darwin_123" 
	cmdDetails := component.Command{Id: "ff_darwin_notfound", Args: cmdArgs, Type: "Shell"}


	err := ExecuteCmd(cmdDetails, channels, nil)

	if err == nil {
		t.Fatalf("Expected an error for command not found (Darwin), got nil")
	}
	
	if !strings.Contains(err.Error(), "executable file not found") && !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Expected error to indicate command not found (Darwin), got: %v", err)
	}

	select {
	case errLine := <-channels.StdErr:
		if !strings.Contains(errLine, "executable file not found") && !strings.Contains(errLine, "no such file or directory") {
			// The error message sent to the channel includes a prefix like "Error starting command: "
			// So, we check if the relevant part is present.
			if !strings.Contains(errLine, "executable file not found") && !strings.Contains(errLine, "no such file or directory") {
				t.Errorf("Expected command not found error on StdErr (Darwin), got: %s", errLine)
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for error on StdErr (Darwin)")
	}

	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID to be sent on command start failure (Darwin), got %d", pid)
	default:
	}
}

func TestFireForget_ExecuteCmd_Darwin_WithInput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "ff_darwin_input", Args: "cat", Type: "Shell"}
	
	inputData := "hello darwin fireforget input" // Input string from prompt's code block
	inPipeReader := io.NopCloser(bytes.NewBufferString(inputData))

	done := make(chan error)
	go func() {
		done <- ExecuteCmd(cmdDetails, channels, inPipeReader)
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("ExecuteCmd failed with input (Darwin): %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("ExecuteCmd did not return quickly with input (Darwin).")
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID with input (Darwin), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID with input (Darwin)")
	}

	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start with input (Darwin), got: %s", errLine)
	default:
	}
}
