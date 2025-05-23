//go:build linux,implant && (!lp || !teamserver)

package firecollect

import (
	"shlyuz/pkg/component"
	"strings"
	"testing"
	"time"
)

func newTestExecChannels() *component.ComponentExecutionChannel { // Copied from generic test
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 10),
		StdErr: make(chan string, 10),
	}
}

func TestFireCollect_Execute_Linux_SuccessWithOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_linux_success_out", Args: "echo hello collect linux", Type: "Shell"}

	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Linux), got: %v. Output: '%s'", err, output)
	}

	expectedOutput := "hello collect linux"
	if strings.TrimSpace(output) != expectedOutput {
		t.Errorf("Expected output '%s' (Linux), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Linux), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Linux)")
	}
}

func TestFireCollect_Execute_Linux_SuccessNoOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_linux_success_no_out", Args: "echo -n", Type: "Shell"}

	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Linux), got: %v", err)
	}

	if output != "" {
		t.Errorf("Expected empty output (Linux), got '%s'", output)
	}
	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Linux), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Linux)")
	}
}

func TestFireCollect_Execute_Linux_FailWithStderrAndOutput(t *testing.T) {
	channels := newTestExecChannels()
	// Corrected quoting for bash -c command
	cmdStr := "bash -c \"echo 'partial linux output'; echo 'linux error message' >&2; exit 1\""
	cmdDetails := component.Command{Id: "fc_linux_fail_stderr", Args: cmdStr, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Linux), got %d", pid)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Linux)")
	}

	if err == nil {
		t.Fatalf("Expected an error (Linux), got nil")
	}
	errorStr := err.Error()
	if !strings.Contains(errorStr, "linux error message") {
		t.Errorf("Expected error to contain stderr 'linux error message' (Linux), got: %s", errorStr)
	}
	if !strings.Contains(errorStr, "exit status 1") { // Linux specific
		t.Errorf("Expected error to contain 'exit status 1' (Linux), got: %s", errorStr)
	}
	
	expectedOutput := "partial linux output"
	if strings.TrimSpace(output) != expectedOutput {
		t.Errorf("Expected partial output '%s' (Linux), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}
}

func TestFireCollect_Execute_Linux_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArgs := "a_linux_collect_cmd_that_truly_does_not_exist_123"
	cmdDetails := component.Command{Id: "fc_linux_cmd_not_found", Args: cmdArgs, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID for command not found (Linux), got %d", pid)
	case <-time.After(200 * time.Millisecond):
		// Good
	}

	if err == nil {
		t.Fatalf("Expected an error for command not found (Linux), got nil")
	}
	errorStr := err.Error()
	// Linux specific error messages for command not found when using os/exec
	if !strings.Contains(errorStr, "executable file not found") && !strings.Contains(errorStr, "no such file or directory") {
		t.Errorf("Expected error to indicate command not found (Linux: 'executable file not found' or 'no such file or directory'), got: %s", errorStr)
	}

	if output != "" {
		t.Errorf("Expected empty output for command not found (Linux), got '%s'", output)
	}
}
