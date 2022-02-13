package core

import "errors"

var (
	ErrHostAlreadyExists = errors.New("host already exists")

	ErrHostNotFound = errors.New("host not found")
)
