//go:build linux,implant && (!lp || !teamserver) // Match build tags of the tested file

package fire

import (
	"shlyuz/pkg/component"
	"strings"
	"sync"
	"testing"
	"time"
	"os/exec" // For ExitError type assertion
)

// Helper function to create a basic execChannels for testing
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 10), // Should remain empty based on current fire_linux.go
		StdErr: make(chan string, 10),
		// Other channels like Done, Cancel can be nil if not used by fire.ExecuteCmd directly
	}
}

func TestExecuteCmd_Success(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_success", Args: "echo hello world", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	// Check StdErr for unexpected errors from Wait()
	// For a successful echo, StdErr should ideally be empty or closed by timeout.
	select {
	case errLine := <-channels.StdErr:
		// It's possible for some systems/shells to emit minor things to stderr even on success.
		// However, for `echo`, we expect it to be clean or specifically not an ExitError.
		// For `echo`, any output to stderr is likely not an *execution* error from `Wait()`.
		// The critical part is that `command.Wait()` itself didn't return an error.
		// The current fire_linux.go sends `err.Error()` from `Wait()` to StdErr.
		// So, if StdErr has something, it must not be indicative of `echo` failing.
		// This test is a bit tricky because fire_linux.go doesn't explicitly signal "success" from Wait().
		// Absence of an error *string* that looks like an ExitError is the best we can do here.
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no 'exit status' error on StdErr for successful command, got: %s", errLine)
		}
		// Allow for other non-critical stderr if any, then timeout
		select {
		case <-channels.StdErr:
		case <-time.After(100 * time.Millisecond):
		}
	case <-time.After(200 * time.Millisecond): // Give time for command to run and potential error to be sent
		// This is good, means no error from Wait() was sent to StdErr.
	}

	// StdOut should be empty
	select {
	case outLine := <-channels.StdOut:
		t.Errorf("Expected StdOut to be empty, got: %s", outLine)
	default:
		// Good, StdOut is empty
	}
}

func TestExecuteCmd_Fail_ExitStatus(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_fail", Args: "false", Type: "Shell"} // 'false' command exits with status 1

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundError := false
	timeout := time.After(500 * time.Millisecond) // Increased timeout
	// Loop to drain potential multiple stderr lines, looking for the specific error
	for {
		select {
		case errLine := <-channels.StdErr:
			// We are looking for the error from command.Wait(), which is an ExitError.
			// The string representation is "exit status X"
			if strings.Contains(errLine, "exit status 1") {
				foundError = true
				break // Found the error we're looking for
			}
			t.Logf("Received on StdErr (and ignoring for this check): %s", errLine)
		case <-timeout:
			t.Errorf("Timeout waiting for exit status error on StdErr")
			return // Exit test on timeout
		}
		if foundError {
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'exit status 1' on StdErr, but not found")
	}
}


func TestExecuteCmd_StdoutDrained(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_stdout_drain", Args: "echo this is stdout", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	select {
	case line := <-channels.StdOut:
		t.Errorf("StdOut channel should be empty as output is drained, but got: %s", line)
	case <-time.After(200 * time.Millisecond):
		// Success path: StdOut is empty after a timeout
	}
	// Ensure no unexpected execution error on StdErr
	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no 'exit status' error on StdErr for stdout drain test, got: %s", errLine)
		}
	default:
		// Good
	}
}

func TestExecuteCmd_StderrOutput(t *testing.T) {
	channels := newTestExecChannels()
	// Using bash -c to reliably redirect to stderr
	cmd := component.Command{Id: "test_stderr_output", Args: "bash -c 'echo this is stderr >&2'", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundStderr := false
	timeout := time.After(500 * time.Millisecond)
	// Loop to drain potential multiple stderr lines
	for {
		select {
		case line := <-channels.StdErr:
			if strings.TrimSpace(line) == "this is stderr" {
				foundStderr = true
				// Keep draining in case there are other messages or the Wait error
			} else if strings.Contains(line, "exit status") {
				t.Errorf("Command itself should succeed, got unexpected exit status error: %s", line)
			}
			// else, other stderr output, just log
			t.Logf("StdErr line: %s", line)

			if foundStderr && !strings.Contains(line, "exit status") { // If we found it and it's not an error status
				// We might get other stderr lines, or the error from Wait() if the command failed.
				// For this test, the command itself succeeds.
			}
		case <-timeout:
			if !foundStderr {
				t.Errorf("Timeout waiting for 'this is stderr' on StdErr")
			}
			return // Exit test on timeout
		}
		// If foundStderr is true, we can break if we are sure no other critical messages (like exit status) will follow.
		// For now, let it drain until timeout to see all messages.
		// A more precise test might involve a WaitGroup for stderr goroutine if possible.
		if foundStderr { // If we found it, we are good.
			break
		}
	}
	if !foundStderr {
		t.Errorf("Expected 'this is stderr' on StdErr, but not found after loop")
	}
}

func TestExecuteCmd_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_cmd_not_found", Args: "a_very_unlikely_command_to_exist_shlyuz_test", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	// PID might or might not be sent depending on when the error occurs (before or after Start)
	// For command not found, Start() itself fails. So, no PID.
	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID for command not found, got %d", pid)
	case <-time.After(100 * time.Millisecond):
		// Good, no PID.
	}

	foundError := false
	timeout := time.After(200 * time.Millisecond)
	// Loop to drain potential multiple stderr lines
	for {
		select {
		case errLine := <-channels.StdErr:
			// Error from Start() for "command not found" is typically "exec: "command_name": executable file not found in $PATH"
			// Error from Wait() if Start somehow succeeded would be an ExitError.
			if strings.Contains(errLine, "executable file not found") || strings.Contains(errLine, "no such file or directory") {
				foundError = true
				break 
			}
			t.Logf("Received on StdErr (and ignoring for this check): %s", errLine)
		case <-timeout:
			t.Errorf("Timeout waiting for command not found error on StdErr")
			return 
		}
		if foundError {
			break
		}
	}
	if !foundError {
		t.Errorf("Expected 'executable file not found' or 'no such file or directory' error on StdErr")
	}
}
