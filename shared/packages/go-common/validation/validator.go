package validation

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule"`
	Value   any    `json:"value"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	messages := make([]string, len(ve))
	for i, e := range ve {
		messages[i] = e.Error()
	}
	return strings.Join(messages, "; ")
}

// HasField checks if any error matches the given field name.
func (ve ValidationErrors) HasField(field string) bool {
	for _, e := range ve {
		if e.Field == field {
			return true
		}
	}
	return false
}

// FieldErrors returns all errors for a specific field.
func (ve ValidationErrors) FieldErrors(field string) []string {
	var errs []string
	for _, e := range ve {
		if e.Field == field {
			errs = append(errs, e.Message)
		}
	}
	return errs
}

// Validator provides struct validation capabilities using reflection.
type Validator struct {
	rules map[string][]Rule
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{
		rules: make(map[string][]Rule),
	}
}

// AddRule adds a validation rule for a specific field.
func (v *Validator) AddRule(field string, rule Rule) *Validator {
	v.rules[field] = append(v.rules[field], rule)
	return v
}

// Validate validates the given struct using the registered rules.
// The struct must be a pointer to a struct.
func (v *Validator) Validate(obj any) ValidationErrors {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ValidationErrors{
			{Field: "_", Message: "value must be a struct or pointer to struct", Rule: "type"},
		}
	}

	var errors ValidationErrors

	for field, rules := range v.rules {
		fieldVal := v.getFieldValue(val, field)
		for _, rule := range rules {
			if err := rule.Validate(fieldVal); err != nil {
				errors = append(errors, &ValidationError{
					Field:   field,
					Message: err.Error(),
					Rule:    rule.Name(),
					Value:   fieldVal.Interface(),
				})
			}
		}
	}

	return errors
}

// ValidateSingle validates a single value against the provided rules.
func (v *Validator) ValidateSingle(value any, rules []Rule) ValidationErrors {
	var errors ValidationErrors
	for _, rule := range rules {
		if err := rule.Validate(reflect.ValueOf(value)); err != nil {
			errors = append(errors, &ValidationError{
				Field:   "value",
				Message: err.Error(),
				Rule:    rule.Name(),
				Value:   value,
			})
		}
	}
	return errors
}

// getFieldValue retrieves the value of a field from a struct using reflection.
func (v *Validator) getFieldValue(val reflect.Value, field string) reflect.Value {
	// Try direct field name match
	if f := val.FieldByName(field); f.IsValid() {
		return f
	}

	// Try snake_case to CamelCase conversion
	camelCase := toCamelCase(field)
	if f := val.FieldByName(camelCase); f.IsValid() {
		return f
	}

	// Try to find field by tag
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("json"); tag == field {
			return val.Field(i)
		}
		if tag := t.Field(i).Tag.Get("form"); tag == field {
			return val.Field(i)
		}
	}

	return reflect.Value{}
}

// toCamelCase converts snake_case to CamelCase.
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
