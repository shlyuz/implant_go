//go:build implant && (!lp || !teamserver)

package fire

import (
	"os/exec"
	"syscall"
)

// No communication, aside from what's returned from the status. Exported function MUST return when the exported function is called
func ExecuteCmd(cmd string) error {
	command := exec.Command(cmd)
	command.SysProcAttr = &syscall.SysProcAttr{}
	command.SysProcAttr.CmdLine = cmd
	err := command.Run()
	// I know we aren't SUPPOSED to return an error, but whatever, that's dumb
	return err
}
