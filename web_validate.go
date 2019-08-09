package bootx

import (
	"github.com/gen-iot/std"
)

type Validator struct {
}

func NewWebValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(i interface{}) error {
	return std.ValidateStruct(i)
}
