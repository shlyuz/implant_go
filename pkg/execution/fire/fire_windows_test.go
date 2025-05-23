//go:build windows,implant && (!lp || !teamserver) // Match build tags

package fire

import (
	"shlyuz/pkg/component"
	"strings"
	// "sync" // Not explicitly used in the provided test structure, but can be added if needed for complex async logic
	"testing"
	"time"
	// "os/exec" // For ExitError type assertion if needed, though string checks are used
)

// Helper function to create a basic execChannels for testing
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 10), // Should remain empty
		StdErr: make(chan string, 10),
	}
}

func TestExecuteCmd_Windows_Success(t *testing.T) {
	channels := newTestExecChannels()
	// cmdDetails.Args is passed to `cmd /C` in fire_windows.go
	cmd := component.Command{Id: "test_win_success", Args: "echo hello world", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no 'exit status' error on StdErr for successful command, got: %s", errLine)
		}
		// Drain any other non-critical stderr
		for i := 0; i < cap(channels.StdErr)-1; i++ { // -1 because we consumed one
			select {
			case <-channels.StdErr:
			default:
				break
			}
		}
	case <-time.After(500 * time.Millisecond): // Increased timeout for Windows
		// Good, no error from Wait()
	}

	select {
	case outLine := <-channels.StdOut:
		t.Errorf("Expected StdOut to be empty, got: %s", outLine)
	default:
	}
}

func TestExecuteCmd_Windows_Fail_ExitStatus(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_win_fail", Args: "exit 1", Type: "Shell"} // cmd /C exit 1

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundError := false
	timeout := time.After(1 * time.Second) // Increased timeout for Windows
	for {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") { // Windows error string for exit code
				foundError = true
				break
			}
			t.Logf("Received on StdErr (ignoring for this check): %s", errLine)
		case <-timeout:
			if !foundError { // Check one last time if timeout occurred before error was found
				t.Errorf("Timeout waiting for exit status error on StdErr. Expected 'exit status 1'.")
			}
			return
		}
		if foundError {
			break
		}
	}
	if !foundError { // This check might be redundant if the loop logic is correct, but good for clarity
		t.Errorf("Expected 'exit status 1' on StdErr, but not found")
	}
}

func TestExecuteCmd_Windows_StdoutDrained(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_win_stdout_drain", Args: "echo this is windows stdout", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	select {
	case line := <-channels.StdOut:
		t.Errorf("StdOut channel should be empty, but got: %s", line)
	case <-time.After(500 * time.Millisecond):
		// Good
	}
	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no 'exit status' error on StdErr for stdout drain test, got: %s", errLine)
		}
	default:
		// Good
	}
}

func TestExecuteCmd_Windows_StderrOutput(t *testing.T) {
	channels := newTestExecChannels()
	// cmd /C "command >&2"
	cmd := component.Command{Id: "test_win_stderr_output", Args: "echo this is windows stderr >&2", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundStderr := false
	timeout := time.After(1 * time.Second) // Increased timeout
	// Loop to drain potential multiple stderr lines, looking for the specific output
	// and ensuring no unexpected exit status error from the command itself.
	for {
		select {
		case line := <-channels.StdErr:
			if strings.TrimSpace(line) == "this is windows stderr" {
				foundStderr = true
				// Don't break yet; there might be an "exit status" line if the command also failed.
				// For this test, `echo >&2` should succeed.
			} else if strings.Contains(line, "exit status") {
				// This command (echo ... >&2) should exit successfully (status 0).
				// If `cmd /C` itself has an issue or the command string is malformed for `cmd /C`,
				// it might result in an exit status from `cmd.exe`.
				t.Errorf("Command itself should succeed, got unexpected exit status error: %s", line)
			}
			t.Logf("StdErr line: %s", line)
		case <-timeout:
			if !foundStderr {
				t.Errorf("Timeout waiting for 'this is windows stderr' on StdErr")
			}
			return
		}
		if foundStderr { // If we found the specific stderr line and haven't seen an exit status error, we're good.
			// We need a way to know if the command finished without error.
			// The current fire_windows.go will send the error from command.Wait() if it's non-nil.
			// If `echo >&2` succeeds, command.Wait() is nil, so nothing more is sent to StdErr from Wait().
			// We can break after finding the desired stderr line, assuming the command itself is simple and successful.
			break 
		}
	}
	if !foundStderr {
		t.Errorf("Expected 'this is windows stderr' on StdErr, but not found after loop")
	}
}

func TestExecuteCmd_Windows_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmdArg := "a_very_unlikely_command_to_exist_shlyuz_win_test"
	cmd := component.Command{Id: "test_win_cmd_not_found", Args: cmdArg, Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID for cmd.exe, got %d", pid)
	}

	foundCmdErrorMsg := false
	foundExitStatus1 := false
	timeout := time.After(1 * time.Second) // Increased timeout

	// Loop to check all stderr messages
	for {
		select {
		case errLine := <-channels.StdErr:
			t.Logf("StdErr (CmdNotFound): %s", errLine)
			// Check for the message from cmd.exe indicating the command was not recognized
			// This message can be locale-dependent. Checking for the command name is a basic heuristic.
			// Example: "'a_very_unlikely_command_to_exist_shlyuz_win_test' is not recognized..."
			if strings.Contains(errLine, cmdArg) && (strings.Contains(errLine, "is not recognized") || strings.Contains(errLine, "n’est pas reconnu")) {
				foundCmdErrorMsg = true
			}
			// Check for the exit status 1 from cmd.exe itself
			if strings.Contains(errLine, "exit status 1") {
				foundExitStatus1 = true
			}
		case <-timeout:
			goto endLoop // Break outer loop on timeout
		}
		// If both conditions are met, we can break early.
		// Otherwise, the loop continues until timeout or channel closes (implicitly by test ending).
		if foundCmdErrorMsg && foundExitStatus1 {
			break
		}
	}
endLoop:

	if !foundCmdErrorMsg {
		t.Errorf("Expected stderr message from cmd.exe indicating command '%s' was not recognized.", cmdArg)
	}
	if !foundExitStatus1 {
		t.Errorf("Expected 'exit status 1' from cmd.exe on StdErr because the inner command was not found.")
	}
}
