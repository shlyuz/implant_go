//go:build windows,implant && (!lp || !teamserver)

package firecollect

import (
	"shlyuz/pkg/component"
	"strings"
	"testing"
	"time"
)

func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 10),
		StdErr: make(chan string, 10),
	}
}

func TestFireCollect_Execute_Windows_SuccessWithOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_win_success_out", Args: "cmd /C echo hello windows collect", Type: "Shell"}


	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Windows), got: %v. Output: '%s'", err, output)
	}

	expectedOutput := "hello windows collect" 
	if strings.TrimSpace(output) != expectedOutput {
		t.Errorf("Expected output '%s' (Windows), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Windows), got %d", pid)
		}
	case <-time.After(500 * time.Millisecond): 
		t.Errorf("Timeout waiting for PID (Windows)")
	}
}

func TestFireCollect_Execute_Windows_SuccessNoOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_win_success_no_out", Args: "cmd /C type nul", Type: "Shell"}

	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Windows), got: %v", err)
	}

	if strings.TrimSpace(output) != "" { 
		t.Errorf("Expected empty output (Windows), got '%s'", output)
	}
	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Windows), got %d", pid)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Windows)")
	}
}

func TestFireCollect_Execute_Windows_FailWithStderrAndOutput(t *testing.T) {
	channels := newTestExecChannels()
	// Correctly quoted command string for cmd /C
	cmdStr := "cmd /C \"echo partial windows output & echo windows error message >&2 & exit 1\""
	cmdDetails := component.Command{Id: "fc_win_fail_stderr", Args: cmdStr, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Windows), got %d", pid)
		}
	case <-time.After(1 * time.Second): 
		t.Errorf("Timeout waiting for PID (Windows)")
	}

	if err == nil {
		t.Fatalf("Expected an error (Windows), got nil")
	}
	errorStr := err.Error()
	if !strings.Contains(errorStr, "windows error message") {
		t.Errorf("Expected error to contain stderr 'windows error message' (Windows), got: %s", errorStr)
	}
	if !strings.Contains(errorStr, "exit status 1") {
		t.Errorf("Expected error to contain 'exit status 1' (Windows), got: %s", errorStr)
	}
	
	expectedOutput := "partial windows output"
	if strings.TrimSpace(output) != expectedOutput { 
		t.Errorf("Expected partial output '%s' (Windows), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}
}

func TestFireCollect_Execute_Windows_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArgs := "a_windows_collect_cmd_that_truly_does_not_exist_XYZ123"
	cmdDetails := component.Command{Id: "fc_win_cmd_not_found", Args: cmdArgs, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID for command not found (Windows), got %d", pid)
	case <-time.After(500 * time.Millisecond): // Increased timeout slightly
		// Good
	}

	if err == nil {
		t.Fatalf("Expected an error for command not found (Windows), got nil")
	}
	errorStr := err.Error()
	if !strings.Contains(errorStr, "executable file not found") && 
	   !strings.Contains(errorStr, "The system cannot find the file specified") { 
		t.Errorf("Expected error to indicate command not found (Windows), got: %s", errorStr)
	}

	if strings.TrimSpace(output) != "" { 
		t.Errorf("Expected empty output for command not found (Windows), got '%s'", output)
	}
}
