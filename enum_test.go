package enum

import (
	"fmt"
	"testing"
)

// @enumGenerated
type gender string

const (
	male    gender = "male"
	female  gender = "female"
	unknown gender = "unknown"
)

func (g gender) Values() []gender {
	return []gender{
		male,
		female,
		unknown,
	}
}

func (g gender) String() string {
	return fmt.Sprintf("%v", g)
}

func TestEnum(t *testing.T) {

}
