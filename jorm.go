package jorm

import (
	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/validator"
)

// Re-export core types and functions
type DB = core.DB
type Query = core.Query
type Options = core.Options

var Open = core.Open

// Re-export validator types and functions
type Validator = validator.Validator
type ValidationErrors = validator.ValidationErrors
type Rules = validator.Rules
type Rule = validator.Rule

var (
	Validate = validator.Validate
	Check    = validator.Check
	FirstMsg = validator.FirstMsg

	// Rules
	Required     = validator.Required
	Email        = validator.Email
	Mobile       = validator.Mobile
	URL          = validator.URL
	IP           = validator.IP
	JSON         = validator.JSON
	UUID         = validator.UUID
	Numeric      = validator.Numeric
	Alpha        = validator.Alpha
	AlphaNumeric = validator.AlphaNumeric
	NoHTML       = validator.NoHTML

	// Rule creators
	MinLen   = validator.MinLen
	MaxLen   = validator.MaxLen
	Range    = validator.Range
	In       = validator.In
	Datetime = validator.Datetime
	Regexp   = validator.Regexp
	Contains = validator.Contains
	Excludes = validator.Excludes
)
