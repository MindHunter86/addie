package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	*validator.Validate
}

func New() *Validator {
	return &Validator{
		Validate: validator.New(),
	}
}

// Var validates value and associates validation errors with key.
func (v *Validator) Var(key string, value any, tag string) error {
	err := v.Validate.Var(value, tag)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}

	result := make(validator.ValidationErrors, len(validationErrors))

	for i, fieldError := range validationErrors {
		result[i] = keyedFieldError{
			FieldError: fieldError,
			key:        key,
		}
	}

	return result
}

type keyedFieldError struct {
	validator.FieldError
	key string
}

func (e keyedFieldError) Namespace() string {
	return joinKey(e.key, e.FieldError.Namespace())
}

func (e keyedFieldError) StructNamespace() string {
	return joinKey(e.key, e.FieldError.StructNamespace())
}

func (e keyedFieldError) Field() string {
	return joinKey(e.key, e.FieldError.Field())
}

func (e keyedFieldError) StructField() string {
	return joinKey(e.key, e.FieldError.StructField())
}

func (e keyedFieldError) Error() string {
	return fmt.Sprintf(
		"Key: '%s' Error:Field validation for '%s' failed on the '%s' tag",
		e.Namespace(),
		e.Field(),
		e.Tag(),
	)
}

func joinKey(key, namespace string) string {
	if namespace == "" {
		return key
	}

	if strings.HasPrefix(namespace, "[") {
		return key + namespace
	}

	return key + "." + namespace
}
