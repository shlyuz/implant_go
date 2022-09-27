package shlyuz

import (
	"log"

	"shlyuz/pkg/utils/logging"
)

func Main() {
	log.SetPrefix(logging.GetLogPrefix())
	log.Println("Started Shlyuz")
}
