package validator

import (
	"github.com/pkg/errors"
	"reflect"
	strconv "strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

var ErrUnexpectedValidate = errors.New("unexpected validate value")
var ErrWrongLen = errors.New("wrong len")
var ErrWrongIn = errors.New("wrong in")
var ErrWrongMin = errors.New("wrong min")
var ErrWrongMax = errors.New("wrong max")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	b := strings.Builder{}
	first := true
	for _, err := range v {
		if first {
			first = false
		} else {
			b.WriteString(". ")
		}
		b.WriteString(err.Err.Error())
	}
	return b.String()
}

const prefixLen = "len:"
const prefixIn = "in:"
const prefixMin = "min:"
const prefixMax = "max:"

func validateInt(validate string, v int) *ValidationError {
	if validate == "" {
		return nil
	}

	if strings.HasPrefix(validate, prefixIn) {
		values := strings.Split(validate[len(prefixIn):], ",")
		if len(values) == 0 || values[0] == "" {
			return &ValidationError{Err: ErrWrongIn}
		}
		for _, strVal := range values {
			val, err := strconv.Atoi(strVal)
			if err != nil {
				return &ValidationError{Err: ErrWrongIn}
			}
			if v == val {
				return nil
			}
		}
		return &ValidationError{Err: ErrWrongIn}
	}
	if strings.HasPrefix(validate, prefixMin) {
		min, err := strconv.Atoi(validate[len(prefixMin):])
		if err != nil {
			return &ValidationError{Err: ErrInvalidValidatorSyntax}
		}
		if v < min {
			return &ValidationError{Err: ErrWrongMin}
		}
		return nil
	}
	if strings.HasPrefix(validate, prefixMax) {
		max, err := strconv.Atoi(validate[len(prefixMax):])
		if err != nil {
			return &ValidationError{Err: ErrInvalidValidatorSyntax}
		}
		if v > max {
			return &ValidationError{Err: ErrWrongMax}
		}
		return nil
	}
	return &ValidationError{Err: ErrUnexpectedValidate}
}

func validateString(validate string, v string) *ValidationError {
	if validate == "" {
		return nil
	}

	if strings.HasPrefix(validate, prefixLen) {
		expLen, err := strconv.Atoi(validate[len(prefixLen):])
		if err != nil {
			return &ValidationError{Err: ErrInvalidValidatorSyntax}
		}
		if len(v) != expLen {
			return &ValidationError{Err: ErrWrongLen}
		}
		return nil
	}
	if strings.HasPrefix(validate, prefixIn) {
		values := strings.Split(validate[len(prefixIn):], ",")
		if len(values) == 0 || values[0] == "" {
			return &ValidationError{Err: ErrWrongIn}
		}
		for _, value := range values {
			if v == value {
				return nil
			}
		}
		return &ValidationError{Err: ErrWrongIn}
	}
	if strings.HasPrefix(validate, prefixMin) {
		min, err := strconv.Atoi(validate[len(prefixMin):])
		if err != nil {
			return &ValidationError{Err: ErrInvalidValidatorSyntax}
		}
		if len(v) < min {
			return &ValidationError{Err: ErrWrongMin}
		}
		return nil
	}
	if strings.HasPrefix(validate, prefixMax) {
		max, err := strconv.Atoi(validate[len(prefixMax):])
		if err != nil {
			return &ValidationError{Err: ErrInvalidValidatorSyntax}
		}
		if len(v) > max {
			return &ValidationError{Err: ErrWrongMax}
		}
		return nil
	}
	return &ValidationError{Err: ErrUnexpectedValidate}
}

func Validate(v any) error {
	if reflect.ValueOf(v).Kind() != reflect.Struct {
		return ErrNotStruct
	}
	valueElem := reflect.ValueOf(&v).Elem().Elem()
	typeElem := reflect.TypeOf(v)

	validationErrors := ValidationErrors{}
	for i := 0; i < valueElem.NumField(); i++ {
		field := typeElem.Field(i)
		validate := field.Tag.Get("validate")
		if !field.IsExported() && validate != "" {
			validationErrors = append(validationErrors, ValidationError{Err: ErrValidateForUnexportedFields})
			continue
		}
		switch field.Type.Kind() {
		case reflect.Int:
			err := validateInt(validate, int(valueElem.Field(i).Int()))
			if err != nil {
				validationErrors = append(validationErrors, *err)
			}
		case reflect.String:
			err := validateString(validate, valueElem.Field(i).String())
			if err != nil {
				validationErrors = append(validationErrors, *err)
			}
		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.Int:
				for ind := 0; ind < valueElem.Field(i).Len(); ind++ {
					err := validateInt(validate, int(valueElem.Field(i).Index(ind).Int()))
					if err != nil {
						validationErrors = append(validationErrors, *err)
					}
				}
			case reflect.String:
				for ind := 0; ind < valueElem.Field(i).Len(); ind++ {
					err := validateString(validate, valueElem.Field(i).Index(ind).String())
					if err != nil {
						validationErrors = append(validationErrors, *err)
					}
				}
			}
		case reflect.Struct:
			if validate == "inner" {
				nestedErr := Validate(valueElem.Field(i).Interface())
				if nestedErr != nil {
					nestedValErr := &ValidationErrors{}
					if errors.As(nestedErr, nestedValErr) {
						validationErrors = append(validationErrors, *nestedValErr...)
					} else {
						validationErrors = append(validationErrors, ValidationError{Err: nestedErr})
					}
				}
			}
		}
	}
	if len(validationErrors) == 0 {
		return nil
	}
	return validationErrors
}
