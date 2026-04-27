package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Rule is an interface for validation rules.
type Rule interface {
	// Validate checks if the value passes the validation rule.
	Validate(value reflect.Value) error
	// Name returns the name of the validation rule.
	Name() string
}

// ruleFunc is a function-based rule implementation.
type ruleFunc struct {
	name string
	fn   func(reflect.Value) error
}

func (r *ruleFunc) Validate(value reflect.Value) error {
	return r.fn(value)
}

func (r *ruleFunc) Name() string {
	return r.name
}

// getStringValue extracts a string value from a reflect.Value.
func getStringValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprintf("%v", v.Interface())
}

// isEmpty checks if a value is empty (zero value).
func isEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

// Required creates a rule that checks if a field is not empty.
func Required() Rule {
	return &ruleFunc{
		name: "required",
		fn: func(v reflect.Value) error {
			if isEmpty(v) {
				return fmt.Errorf("field is required")
			}
			return nil
		},
	}
}

// Email creates a rule that validates an email format.
func Email() Rule {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return &ruleFunc{
		name: "email",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil // Use Required() for non-empty validation
			}
			if !emailRegex.MatchString(s) {
				return fmt.Errorf("must be a valid email address")
			}
			return nil
		},
	}
}

// MinLength creates a rule that validates minimum string length.
func MinLength(min int) Rule {
	return &ruleFunc{
		name: "min_length",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil
			}
			if utf8.RuneCountInString(s) < min {
				return fmt.Errorf("must be at least %d characters long", min)
			}
			return nil
		},
	}
}

// MaxLength creates a rule that validates maximum string length.
func MaxLength(max int) Rule {
	return &ruleFunc{
		name: "max_length",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil
			}
			if utf8.RuneCountInString(s) > max {
				return fmt.Errorf("must be at most %d characters long", max)
			}
			return nil
		},
	}
}

// Between creates a rule that validates if a numeric value is within a range.
func Between(min, max int) Rule {
	return &ruleFunc{
		name: "between",
		fn: func(v reflect.Value) error {
			if isEmpty(v) {
				return nil
			}
			var val int64
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val = v.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val = int64(v.Uint())
			case reflect.Float32, reflect.Float64:
				val = int64(v.Float())
			default:
				return fmt.Errorf("between rule requires a numeric value")
			}
			if val < int64(min) || val > int64(max) {
				return fmt.Errorf("must be between %d and %d", min, max)
			}
			return nil
		},
	}
}

// Match creates a rule that validates if a string matches a regex pattern.
func Match(pattern string, message string) Rule {
	re := regexp.MustCompile(pattern)
	if message == "" {
		message = fmt.Sprintf("must match pattern: %s", pattern)
	}
	return &ruleFunc{
		name: "match",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil
			}
			if !re.MatchString(s) {
				return fmt.Errorf("%s", message)
			}
			return nil
		},
	}
}

// Min creates a rule that validates if a numeric value is at least min.
func Min(min int64) Rule {
	return &ruleFunc{
		name: "min",
		fn: func(v reflect.Value) error {
			if isEmpty(v) {
				return nil
			}
			var val int64
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val = v.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val = int64(v.Uint())
			default:
				return fmt.Errorf("min rule requires a numeric value")
			}
			if val < min {
				return fmt.Errorf("must be at least %d", min)
			}
			return nil
		},
	}
}

// Max creates a rule that validates if a numeric value is at most max.
func Max(max int64) Rule {
	return &ruleFunc{
		name: "max",
		fn: func(v reflect.Value) error {
			if isEmpty(v) {
				return nil
			}
			var val int64
			switch v.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val = v.Int()
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val = int64(v.Uint())
			default:
				return fmt.Errorf("max rule requires a numeric value")
			}
			if val > max {
				return fmt.Errorf("must be at most %d", max)
			}
			return nil
		},
	}
}

// URL creates a rule that validates a URL format.
func URL() Rule {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return &ruleFunc{
		name: "url",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil
			}
			if !urlRegex.MatchString(s) {
				return fmt.Errorf("must be a valid URL")
			}
			return nil
		},
	}
}

// In creates a rule that validates if a value is in a list of allowed values.
func In(allowed ...string) Rule {
	return &ruleFunc{
		name: "in",
		fn: func(v reflect.Value) error {
			s := getStringValue(v)
			if s == "" {
				return nil
			}
			for _, a := range allowed {
				if strings.EqualFold(s, a) {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
		},
	}
}
