package model

import "errors"

var ErrDuplicateemail = errors.New("duplicate key")

var ErrecordNotFound = errors.New("no record found")

var ErrInvalidCredentials = errors.New("invalid credentials")
