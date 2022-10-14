package instructions

import (
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/utils/idgen"
	"shlyuz/pkg/utils/uname"
	"time"
)

type Transaction struct {
	ComponentId string
	Cmd         string
	Arg         []byte
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

type EventHist struct {
	Timestamp   string
	Event       string
	ComponentId string
}

type CmdOutput struct {
	State        string
	EventHistory EventHist
	Ipk          asymmetric.PublicKey
}

// Create an instruction frame from a passed data frame
//
// @param DataFrame: A dataframe to create an instruction from
func CreateInstructionFrame(DataFrame Transaction) *InstructionFrame {
	IFrame := new(InstructionFrame)
	IFrame.ComponentId = DataFrame.ComponentId
	IFrame.Cmd = DataFrame.Cmd
	IFrame.Date = time.Now().String()
	if DataFrame.TxId != "" {
		IFrame.TxId = DataFrame.TxId
	} else {
		IFrame.TxId = idgen.GenerateTxId()
	}
	IFrame.Uname = *uname.GetUname()
	return IFrame
}
