package core

import (
	"fmt"
)

// JormError represents a structured error in the ORM
type JormError struct {
	Code    int
	Message string
	Err     error
}

func (e *JormError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *JormError) Unwrap() error { return e.Err }

const (
	ErrCodeNotFound = iota + 1
	ErrCodeInvalidModel
	ErrCodeDuplicateKey
	ErrCodeForeignKey
	ErrCodeConnectionFailed
	ErrCodeTransactionAborted
	ErrCodeInvalidSQL
)

var (
	ErrRecordNotFound     = &JormError{Code: ErrCodeNotFound, Message: "record not found"}
	ErrInvalidModel       = &JormError{Code: ErrCodeInvalidModel, Message: "invalid model"}
	ErrDuplicateKey       = &JormError{Code: ErrCodeDuplicateKey, Message: "duplicate key"}
	ErrForeignKey         = &JormError{Code: ErrCodeForeignKey, Message: "foreign key constraint"}
	ErrConnectionFailed   = &JormError{Code: ErrCodeConnectionFailed, Message: "connection failed"}
	ErrTransactionAborted = &JormError{Code: ErrCodeTransactionAborted, Message: "transaction aborted"}
	ErrInvalidSQL         = &JormError{Code: ErrCodeInvalidSQL, Message: "invalid sql"}
)
