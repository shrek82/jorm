package core

import (
	"errors"
)

var (
	// ErrRecordNotFound is returned when a query expects at least one record but none were found.
	ErrRecordNotFound = errors.New("record not found")
	// ErrModelNotFound is returned when a model metadata cannot be found for a given type.
	ErrModelNotFound = errors.New("model not found")
	// ErrInvalidModel is returned when a model definition is invalid (e.g., missing primary key).
	ErrInvalidModel = errors.New("invalid model")
	// ErrInvalidQuery is returned when a query is malformed or cannot be executed.
	ErrInvalidQuery = errors.New("invalid query")
	// ErrRelationNotFound is returned when a requested relation does not exist on the model.
	ErrRelationNotFound = errors.New("relation not found")
	// ErrDuplicateKey is returned when a database unique constraint is violated.
	ErrDuplicateKey = errors.New("duplicate key")
	// ErrForeignKey is returned when a database foreign key constraint is violated.
	ErrForeignKey = errors.New("foreign key constraint")
	// ErrConnectionFailed is returned when the database connection cannot be established or is lost.
	ErrConnectionFailed = errors.New("connection failed")
	// ErrInvalidSQL is returned when a raw SQL statement is empty or malformed.
	ErrInvalidSQL = errors.New("invalid sql")
)
