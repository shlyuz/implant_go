package routine

import (
	"shlyuz/pkg/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/symmetric"
)

type DataFrame struct {
	component_id string
	cmd          string
	args         []string
}

func genInit()
