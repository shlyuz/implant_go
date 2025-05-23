//go:build darwin,implant && (!lp || !teamserver)

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

func TestFireCollect_Execute_Darwin_SuccessWithOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_darwin_success_out", Args: "echo hello collect darwin", Type: "Shell"}

	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Darwin), got: %v. Output: '%s'", err, output)
	}

	expectedOutput := "hello collect darwin" 
	if strings.TrimSpace(output) != expectedOutput {
		t.Errorf("Expected output '%s' (Darwin), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Darwin), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Darwin)")
	}
}

func TestFireCollect_Execute_Darwin_SuccessNoOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmdDetails := component.Command{Id: "fc_darwin_success_no_out", Args: "echo -n", Type: "Shell"}

	output, err := Execute(cmdDetails, channels)
	if err != nil {
		t.Fatalf("Expected no error (Darwin), got: %v", err)
	}

	if output != "" {
		t.Errorf("Expected empty output (Darwin), got '%s'", output)
	}
	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Darwin), got %d", pid)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Darwin)")
	}
}

func TestFireCollect_Execute_Darwin_FailWithStderrAndOutput(t *testing.T) {
	channels := newTestExecChannels()
	// Correctly quoted command string for bash -c
	cmdStr := "bash -c \"echo 'partial darwin output'; echo 'darwin error message' >&2; exit 1\""
	cmdDetails := component.Command{Id: "fc_darwin_fail_stderr", Args: cmdStr, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		if pid <= 0 {
			t.Errorf("Expected valid PID (Darwin), got %d", pid)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("Timeout waiting for PID (Darwin)")
	}

	if err == nil {
		t.Fatalf("Expected an error (Darwin), got nil")
	}
	errorStr := err.Error()
	if !strings.Contains(errorStr, "darwin error message") {
		t.Errorf("Expected error to contain stderr 'darwin error message' (Darwin), got: %s", errorStr)
	}
	if !strings.Contains(errorStr, "exit status 1") { // Darwin specific (same as Linux)
		t.Errorf("Expected error to contain 'exit status 1' (Darwin), got: %s", errorStr)
	}
	
	expectedOutput := "partial darwin output"
	if strings.TrimSpace(output) != expectedOutput {
		t.Errorf("Expected partial output '%s' (Darwin), got '%s' (Original: '%s')", expectedOutput, strings.TrimSpace(output), output)
	}
}

func TestFireCollect_Execute_Darwin_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArgs := "a_darwin_collect_cmd_that_truly_does_not_exist_456"
	cmdDetails := component.Command{Id: "fc_darwin_cmd_not_found", Args: cmdArgs, Type: "Shell"}

	output, err := Execute(cmdDetails, channels)

	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID for command not found (Darwin), got %d", pid)
	case <-time.After(200 * time.Millisecond):
		// Good
	}

	if err == nil {
		t.Fatalf("Expected an error for command not found (Darwin), got nil")
	}
	errorStr := err.Error()
	if !strings.Contains(errorStr, "executable file not found") && !strings.Contains(errorStr, "no such file or directory") {
		t.Errorf("Expected error to indicate command not found (Darwin: 'executable file not found' or 'no such file or directory'), got: %s", errorStr)
	}

	if output != "" {
		t.Errorf("Expected empty output for command not found (Darwin), got '%s'", output)
	}
}
