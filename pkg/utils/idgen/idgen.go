package idgen

import (
	"github.com/google/uuid"
)

// Generate an Id using uuid
func GenerateId() string {
	InstructionId := uuid.New()
	return InstructionId.String()
}
