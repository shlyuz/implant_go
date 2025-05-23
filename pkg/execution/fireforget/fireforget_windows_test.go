//go:build windows,implant && (!lp || !teamserver)

package fireforget

import (
	"bytes" // For inPipe
	"io"    // For io.NopCloser
	"shlyuz/pkg/component"
	// "strings" // Not strictly needed for these tests
	"testing"
	"time"
)

// Helper function
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 1), // Not used by fireforget
		StdErr: make(chan string, 5), // Should only receive errors from Start() itself
	}
}

func TestFireForget_ExecuteCmd_Windows_Success(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "ff_win_success", Args: "timeout /T 1 /NOBREAK", Type: "Shell"}

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
	case <-time.After(200 * time.Millisecond): // Should return much faster than 1 sec
		t.Fatalf("ExecuteCmd did not return quickly, possibly waiting for command.")
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID, got %d", pid)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timeout waiting for PID")
	}

	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start, got: %s", errLine)
	default:
	}
}

func TestFireForget_ExecuteCmd_Windows_InnerCommandFail(t *testing.T) {
	channels := newTestExecChannels()
	cmdArg := "a_fireforget_win_cmd_that_is_very_unlikely_to_exist"
	cmdDetails := component.Command{Id: "ff_win_innerfail", Args: cmdArg, Type: "Shell"}

	err := ExecuteCmd(cmdDetails, channels, nil)
	if err != nil {
		t.Fatalf("ExecuteCmd returned an error during Start(): %v. This is unexpected for this test case, as cmd.exe itself should start.", err)
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID for cmd.exe, got %d", pid)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timeout waiting for PID")
	}

	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected execChannels.StdErr to be empty as cmd.exe started successfully, got: %s", errLine)
	default:
		// Good, no error from Start() of cmd.exe was sent here.
		// The error from the inner command (not found) is logged by goroutines in fireforget_windows.go.
	}
}

func TestFireForget_ExecuteCmd_Windows_WithInput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "ff_win_input", Args: "findstr /N /P \"\"", Type: "Shell"} // Ensure quotes for findstr args
	
	inputData := "hello windows fireforget input\r\n" // Windows typically uses CRLF
	inPipeReader := io.NopCloser(bytes.NewBufferString(inputData))

	done := make(chan error)
	go func() {
		done <- ExecuteCmd(cmdDetails, channels, inPipeReader)
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			t.Fatalf("ExecuteCmd failed with input: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("ExecuteCmd did not return quickly with input.")
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID with input, got %d", pid)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timeout waiting for PID with input")
	}

	select {
	case errLine := <-channels.StdErr:
		t.Errorf("Expected StdErr to be empty for successful start with input, got: %s", errLine)
	default:
	}
}
