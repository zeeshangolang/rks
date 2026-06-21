package validator

import (
	"regexp"
	"unicode/utf8"
)

type Validator struct {
	FieldErrors   map[string]string
	NonFieldError []string
}

var EmailRx = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

var Usernamerx = regexp.MustCompile(`^[A-Za-z]+$`)

func (v *Validator) Valid() bool {
	return v.FieldErrors == nil && v.NonFieldError == nil
}

func (v *Validator) Addnonfielderror(message string) {
	v.NonFieldError = append(v.NonFieldError, message)
}

func (v *Validator) AddFieldError(key, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}
	if _, ok := v.FieldErrors[key]; !ok {
		v.FieldErrors[key] = message
	}
}

func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}
func Notblank(val string) bool {
	return val != ""
}

func MatchEmail(rx *regexp.Regexp, email string) bool {
	return rx.MatchString(email)
}

func MatchUserName(rx *regexp.Regexp, username string) bool {
	return rx.MatchString(username)
}

func MinChar(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

func MaxChar(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

func PermittedValues[T comparable](Value T, permittedValues ...T) bool {
	for i, _ := range permittedValues {
		if Value == permittedValues[i] {
			return true
		}
	}
	return false
}
