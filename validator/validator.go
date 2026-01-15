package validator

import (
	"fmt"
	"reflect"
	"strings"
)

// Validator is a function that validates a value and returns an error.
type Validator func(value any) error

// ValidationErrors is a map of field names to their validation errors.
type ValidationErrors map[string][]error

func (v ValidationErrors) Error() string {
	var sb strings.Builder
	for field, errs := range v {
		for _, err := range errs {
			if sb.Len() > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v", field, err))
		}
	}
	return sb.String()
}

// Rule is the interface for a single validation rule.
type Rule interface {
	Validate(value any) error
	Msg(msg string) Rule
	Optional() Rule
	When(fn func(value any) bool) Rule
}

// BaseRule provides common functionality for all rules.
type BaseRule struct {
	msg      string
	optional bool
	when     func(value any) bool
}

func (r *BaseRule) SetMsg(msg string) {
	r.msg = msg
}

func (r *BaseRule) SetOptional() {
	r.optional = true
}

func (r *BaseRule) SetWhen(fn func(value any) bool) {
	r.when = fn
}

// ShouldValidate checks if the rule should be executed based on optional and when conditions.
func (r *BaseRule) ShouldValidate(value any) bool {
	if r.when != nil && !r.when(value) {
		return false
	}
	if r.optional {
		return !isZeroValue(value)
	}
	return true
}

// FormatError returns the custom message if set, otherwise returns the default error.
func (r *BaseRule) FormatError(defaultErr error) error {
	if r.msg != "" {
		return fmt.Errorf(r.msg)
	}
	return defaultErr
}

func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return reflect.DeepEqual(v, reflect.Zero(rv.Type()).Interface())
}

// Rules is a map of field names to validation rules.
type Rules map[string][]Rule

// Validate implements the Validator interface for Rules.
func (r Rules) Validate(value any) error {
	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("validator: value must be a struct or pointer to struct")
	}

	errors := make(ValidationErrors)

	for fieldName, rules := range r {
		field := rv.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		val := field.Interface()
		for _, rule := range rules {
			if err := rule.Validate(val); err != nil {
				errors[fieldName] = append(errors[fieldName], err)
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// Validate is a standalone validation function.
func Validate(value any, validators ...Validator) error {
	for _, validator := range validators {
		if err := validator(value); err != nil {
			return err
		}
	}
	return nil
}
