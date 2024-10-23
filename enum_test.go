package enum

import (
	"testing"
)

// @enumGenerated
type gender string

const (
	male    gender = "male"
	female  gender = "female"
	unknown gender = "unknown"
)

func (g gender) Values() []string {
	return []string{
		male.String(),
		female.String(),
		unknown.String(),
	}
}

func (g gender) String() string {
	return string(g)
}

func TestEnum(t *testing.T) {

}
