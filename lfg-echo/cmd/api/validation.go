package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	govalidator "github.com/go-playground/validator/v10"
)

type requestValidator struct {
	v *govalidator.Validate
}

func newRequestValidator() *requestValidator {
	v := govalidator.New()

	// Domain rule: year cannot be in the future.
	_ = v.RegisterValidation("maxyear", func(fl govalidator.FieldLevel) bool {
		switch value := fl.Field().Interface().(type) {
		case int:
			return value <= time.Now().Year()
		case int32:
			return int(value) <= time.Now().Year()
		default:
			return false
		}
	})

	// Domain rule: duplicate genres are not allowed.
	_ = v.RegisterValidation("uniquegenres", func(fl govalidator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.Slice {
			return false
		}

		seen := make(map[string]struct{}, field.Len())
		for i := 0; i < field.Len(); i++ {
			genre := field.Index(i).String()
			if _, exists := seen[genre]; exists {
				return false
			}
			seen[genre] = struct{}{}
		}

		return true
	})

	return &requestValidator{v: v}
}

func (rv *requestValidator) Validate(i any) error {
	return rv.v.Struct(i)
}

func validationMessage(fe govalidator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "maxyear":
		return "cannot be in the future"
	case "uniquegenres":
		return "must not contain duplicate values"
	default:
		return "is invalid"
	}
}

func normalizeFieldName(name string) string {
	if name == "" {
		return "field"
	}

	return strings.ToLower(name[:1]) + name[1:]
}
