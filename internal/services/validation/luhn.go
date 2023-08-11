package validation

import "github.com/ShiraazMoollatjie/goluhn"

func LuhnValidate(value string) error {
	return goluhn.Validate(value)
}
