# enum

# how to use

## import

```
go get github.com/svc0a/enum
```

## tag and generate

```
// @enumGenerated
type gender string

const (
	male    gender = "male"
	female  gender = "female"
	unknown gender = "unknown"
)

func TestEnum(t *testing.T) {
	gen.Generate("enum_test.go")
}
```

## examples

```
// @enumGenerated
type gender string

func (g gender) Values() []string {
	return []string{male.String(), female.String(), unknown.String()}
}
func (g gender) String() string {
	return string(g)
}

const (
	male    gender = "male"
	female  gender = "female"
	unknown gender = "unknown"
)

```