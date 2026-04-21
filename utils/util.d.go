package utils

type ValidatorConfig struct {
	NotEmpty       bool
	MinLength      int
	MaxLength      int
	ExpectedValues []string
}
