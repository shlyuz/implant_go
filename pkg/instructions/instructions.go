package instructions

import (
	"shlyuz/pkg/utils/idgen"
	"shlyuz/pkg/utils/uname"
	"time"
)

type Transaction struct {
	ComponentId string
	Cmd         string
	Arg         string
	TxId        string
}

type InstructionFrame struct {
	ComponentId string
	Cmd         string
	CmdArgs     string
	Date        string
	TxId        string
	Uname       uname.PlatformInfo
}

// Get the platform name

// Create an instruction frame from a passed data frame
func CreateInstructionFrame(DataFrame Transaction) *InstructionFrame {
	IFrame := new(InstructionFrame)
	IFrame.ComponentId = DataFrame.ComponentId
	IFrame.Cmd = DataFrame.Cmd
	IFrame.Date = time.Now().String()
	if DataFrame.TxId != "" {
		IFrame.TxId = DataFrame.TxId
	} else {
		IFrame.TxId = idgen.GenerateId()
	}
	IFrame.Uname = *uname.GetUname()
	return IFrame
}
