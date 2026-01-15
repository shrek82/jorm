package core

import (
	"errors"
)

var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrModelNotFound    = errors.New("model not found")
	ErrInvalidModel     = errors.New("invalid model")
	ErrInvalidQuery     = errors.New("invalid query")
	ErrRelationNotFound = errors.New("relation not found")
	ErrDuplicateKey     = errors.New("duplicate key")
	ErrForeignKey       = errors.New("foreign key constraint")
	ErrConnectionFailed = errors.New("connection failed")
	ErrInvalidSQL       = errors.New("invalid sql")
)
