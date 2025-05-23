//go:build darwin,implant && (!lp || !teamserver) // Key change: Darwin build tag

package fire

import (
	"shlyuz/pkg/component"
	"strings"
	"sync" // sync is used by fire_darwin.go indirectly if execChannels has WaitGroup, not directly by tests here.
	"testing"
	"time"
	"os/exec" // For ExitError type assertion, though not strictly used by current string checks
)

// Helper function to create a basic execChannels for testing
func newTestExecChannels() *component.ComponentExecutionChannel {
	return &component.ComponentExecutionChannel{
		Pid:    make(chan int, 1),
		StdOut: make(chan string, 10), // Should remain empty
		StdErr: make(chan string, 10),
	}
}

func TestExecuteCmd_Darwin_Success(t *testing.T) { // Renamed for clarity
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_darwin_success", Args: "echo hello world", Type: "Shell"}

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
		// Drain other potential non-error stderr messages
		for i := 0; i < cap(channels.StdErr)-1 ; i++ { // -1 because one item might have been consumed
            select {
            case logMsg := <-channels.StdErr:
				t.Logf("Drained additional stderr message (Darwin Success): %s", logMsg)
            default:
                break
            }
        }
	case <-time.After(200 * time.Millisecond): 
		// Good
	}

	select {
	case outLine := <-channels.StdOut:
		t.Errorf("Expected StdOut to be empty, got: %s", outLine)
	default:
	}
}

func TestExecuteCmd_Darwin_Fail_ExitStatus(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_darwin_fail", Args: "false", Type: "Shell"} 

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundError := false
	timeout := time.After(500 * time.Millisecond) 
	for {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "exit status 1") {
				foundError = true
				break 
			}
			t.Logf("Received on StdErr (Darwin, ignoring for this check): %s", errLine)
		case <-timeout:
			t.Logf("Timeout waiting for exit status error on StdErr (Darwin)") // Log before Errorf
			if !foundError {
                 t.Errorf("Final check (Darwin): Expected 'exit status 1' on StdErr, but not found")
            }
			return 
		}
		if foundError {
			break
		}
	}
	if !foundError { // This check is good for clarity even if loop logic is perfect
		t.Errorf("Expected 'exit status 1' on StdErr (Darwin)")
	}
}


func TestExecuteCmd_Darwin_StdoutDrained(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_darwin_stdout_drain", Args: "echo this is darwin stdout", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	select {
	case line := <-channels.StdOut:
		t.Errorf("StdOut channel should be empty (Darwin), but got: %s", line)
	case <-time.After(200 * time.Millisecond):
		// Success
	}
	select {
	case errLine := <-channels.StdErr:
		if strings.Contains(errLine, "exit status") {
			t.Errorf("Expected no 'exit status' error on StdErr for stdout drain test (Darwin), got: %s", errLine)
		}
	default:
	}
}

func TestExecuteCmd_Darwin_StderrOutput(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_darwin_stderr_output", Args: "bash -c 'echo this is darwin stderr >&2'", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	pid := <-channels.Pid
	if pid <= 0 {
		t.Errorf("Expected a valid PID, got %d", pid)
	}

	foundStderr := false
	timeout := time.After(500 * time.Millisecond)
	// Loop to consume all messages from StdErr or until timeout
	for { 
		select {
		case line := <-channels.StdErr:
			t.Logf("StdErr line (Darwin StderrOutput Test): %s", line) // Log every line received
			if strings.TrimSpace(line) == "this is darwin stderr" {
				foundStderr = true
			} else if strings.Contains(line, "exit status") {
				// This specific command should succeed. If it fails, it's an issue.
				t.Errorf("Command itself should succeed (Darwin), got unexpected exit status error: %s", line)
			}
		case <-timeout:
			if !foundStderr {
				t.Errorf("Timeout waiting for 'this is darwin stderr' on StdErr (Darwin)")
			}
			return // Exit the test function
		}
		// Check if we found what we needed and can break, or if we should continue draining until timeout
		// If foundStderr is true, we could potentially break if we are sure no other critical messages (like an unexpected exit status) will follow.
		// However, draining until timeout is safer to catch all stderr lines.
		// For this test, if we see the desired output and no "exit status" error, it's a pass.
		// The loop will continue until timeout to allow logging of all stderr lines.
		// Let's break if found and no error is flagged to make test faster if successful.
		if foundStderr { // If we found the target line, we can break this loop.
			// Check one last time for any immediate error that might have come with it or right after.
			select {
			case line := <-channels.StdErr:
				if strings.Contains(line, "exit status") {
					t.Errorf("Command succeeded in producing stderr, but also an exit status error: %s", line)
				} else {
					t.Logf("Additional stderr after target (Darwin StderrOutput Test): %s", line)
				}
			default:
				// no other immediate error
			}
			break // Exit the for loop
		}
	}
	if !foundStderr {
		t.Errorf("Expected 'this is darwin stderr' on StdErr (Darwin), but not found after loop")
	}
}

func TestExecuteCmd_Darwin_CommandNotFound(t *testing.T) {
	channels := newTestExecChannels()
	cmd := component.Command{Id: "test_darwin_cmd_not_found", Args: "a_very_unlikely_command_to_exist_shlyuz_darwin_test", Type: "Shell"}

	ExecuteCmd(cmd, channels)

	select {
	case pid := <-channels.Pid:
		t.Errorf("Expected no PID for command not found (Darwin), got %d", pid)
	case <-time.After(100 * time.Millisecond):
		// Good
	}

	foundError := false
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case errLine := <-channels.StdErr:
			if strings.Contains(errLine, "executable file not found") || strings.Contains(errLine, "no such file or directory") {
				foundError = true
				break 
			}
			t.Logf("Received on StdErr (Darwin, ignoring for this check): %s", errLine)
		case <-timeout:
			t.Logf("Timeout waiting for command not found error on StdErr (Darwin)") // Log before Errorf
			if !foundError {
                 t.Errorf("Final check (Darwin): Expected 'executable file not found' on StdErr")
            }
			return 
		}
		if foundError {
			break
		}
	}
	if !foundError {
		t.Errorf("Expected 'executable file not found' or 'no such file or directory' error on StdErr (Darwin)")
	}
}
