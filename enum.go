package enum

type Enum interface {
	Values() []string
	String() string
}
