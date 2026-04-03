package utils

import "github.com/go-playground/validator/v10"

var validate = validator.New()

// ValidateStruct validates a struct using go-playground/validator/v10.
func ValidateStruct(data interface{}) error {
	return validate.Struct(data)
}
