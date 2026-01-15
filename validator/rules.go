package validator

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	mobileRegex   = regexp.MustCompile(`^1[3-9]\d{9}$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	numericRegex  = regexp.MustCompile(`^[0-9]+$`)
	alphaRegex    = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	htmlRegex     = regexp.MustCompile(`<[^>]*>`)
)

// --- Required ---

type requiredRule struct {
	BaseRule
}

func (r *requiredRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	if isZeroValue(v) {
		return r.FormatError(fmt.Errorf("is required"))
	}
	return nil
}

func (r *requiredRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *requiredRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *requiredRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var Required Rule = &requiredRule{}

// --- MinLen ---

type minLenRule struct {
	BaseRule
	min int
}

func (r *minLenRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if len(s) < r.min {
		return r.FormatError(fmt.Errorf("length must be at least %d", r.min))
	}
	return nil
}

func (r *minLenRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *minLenRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *minLenRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func MinLen(min int) Rule {
	return &minLenRule{min: min}
}

// --- MaxLen ---

type maxLenRule struct {
	BaseRule
	max int
}

func (r *maxLenRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if len(s) > r.max {
		return r.FormatError(fmt.Errorf("length must be at most %d", r.max))
	}
	return nil
}

func (r *maxLenRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *maxLenRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *maxLenRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func MaxLen(max int) Rule {
	return &maxLenRule{max: max}
}

// --- Range ---

type rangeRule struct {
	BaseRule
	min, max float64
}

func (r *rangeRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	val := reflectToFloat(v)
	if val < r.min || val > r.max {
		return r.FormatError(fmt.Errorf("value must be between %v and %v", r.min, r.max))
	}
	return nil
}

func (r *rangeRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *rangeRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *rangeRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func Range(min, max float64) Rule {
	return &rangeRule{min: min, max: max}
}

func reflectToFloat(v any) float64 {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	}
	return 0
}

// --- In ---

type inRule struct {
	BaseRule
	values []any
}

func (r *inRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	for _, val := range r.values {
		if val == v {
			return nil
		}
	}
	return r.FormatError(fmt.Errorf("value is not in the allowed list"))
}

func (r *inRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *inRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *inRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func In(values ...any) Rule {
	return &inRule{values: values}
}

// --- Email ---

type emailRule struct {
	BaseRule
}

func (r *emailRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !emailRegex.MatchString(strings.ToLower(s)) {
		return r.FormatError(fmt.Errorf("invalid email format"))
	}
	return nil
}

func (r *emailRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *emailRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *emailRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var Email Rule = &emailRule{}

// --- Mobile ---

type mobileRule struct {
	BaseRule
}

func (r *mobileRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !mobileRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("invalid mobile format"))
	}
	return nil
}

func (r *mobileRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *mobileRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *mobileRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var Mobile Rule = &mobileRule{}

// --- URL ---

type urlRule struct {
	BaseRule
}

func (r *urlRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return r.FormatError(fmt.Errorf("invalid URL format"))
	}
	_, err := url.ParseRequestURI(s)
	if err != nil {
		return r.FormatError(fmt.Errorf("invalid URL format"))
	}
	return nil
}

func (r *urlRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *urlRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *urlRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var URL Rule = &urlRule{}

// --- IP ---

type ipRule struct {
	BaseRule
}

func (r *ipRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || net.ParseIP(s) == nil {
		return r.FormatError(fmt.Errorf("invalid IP format"))
	}
	return nil
}

func (r *ipRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *ipRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *ipRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var IP Rule = &ipRule{}

// --- JSON ---

type jsonRule struct {
	BaseRule
}

func (r *jsonRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !json.Valid([]byte(s)) {
		return r.FormatError(fmt.Errorf("invalid JSON format"))
	}
	return nil
}

func (r *jsonRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *jsonRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *jsonRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var JSON Rule = &jsonRule{}

// --- UUID ---

type uuidRule struct {
	BaseRule
}

func (r *uuidRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !uuidRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("invalid UUID format"))
	}
	return nil
}

func (r *uuidRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *uuidRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *uuidRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var UUID Rule = &uuidRule{}

// --- Datetime ---

type datetimeRule struct {
	BaseRule
	format string
}

func (r *datetimeRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return r.FormatError(fmt.Errorf("invalid datetime format"))
	}
	_, err := time.Parse(r.format, s)
	if err != nil {
		return r.FormatError(fmt.Errorf("invalid datetime format, expected %s", r.format))
	}
	return nil
}

func (r *datetimeRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *datetimeRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *datetimeRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func Datetime(format string) Rule {
	return &datetimeRule{format: format}
}

// --- Regexp ---

type regexpRule struct {
	BaseRule
	pattern *regexp.Regexp
}

func (r *regexpRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !r.pattern.MatchString(s) {
		return r.FormatError(fmt.Errorf("does not match pattern"))
	}
	return nil
}

func (r *regexpRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *regexpRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *regexpRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func Regexp(pattern string) Rule {
	return &regexpRule{pattern: regexp.MustCompile(pattern)}
}

// --- Numeric ---

type numericRule struct {
	BaseRule
}

func (r *numericRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !numericRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("must be numeric"))
	}
	return nil
}

func (r *numericRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *numericRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *numericRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var Numeric Rule = &numericRule{}

// --- Alpha ---

type alphaRule struct {
	BaseRule
}

func (r *alphaRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !alphaRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("must be alpha"))
	}
	return nil
}

func (r *alphaRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *alphaRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *alphaRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var Alpha Rule = &alphaRule{}

// --- AlphaNumeric ---

type alphaNumericRule struct {
	BaseRule
}

func (r *alphaNumericRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !alphaNumRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("must be alphanumeric"))
	}
	return nil
}

func (r *alphaNumericRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *alphaNumericRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *alphaNumericRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var AlphaNumeric Rule = &alphaNumericRule{}

// --- Contains ---

type containsRule struct {
	BaseRule
	substr string
}

func (r *containsRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || !strings.Contains(s, r.substr) {
		return r.FormatError(fmt.Errorf("must contain %s", r.substr))
	}
	return nil
}

func (r *containsRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *containsRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *containsRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func Contains(substr string) Rule {
	return &containsRule{substr: substr}
}

// --- Excludes ---

type excludesRule struct {
	BaseRule
	substr string
}

func (r *excludesRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || strings.Contains(s, r.substr) {
		return r.FormatError(fmt.Errorf("must not contain %s", r.substr))
	}
	return nil
}

func (r *excludesRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *excludesRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *excludesRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

func Excludes(substr string) Rule {
	return &excludesRule{substr: substr}
}

// --- NoHTML ---

type noHTMLRule struct {
	BaseRule
}

func (r *noHTMLRule) Validate(v any) error {
	if !r.ShouldValidate(v) {
		return nil
	}
	s, ok := v.(string)
	if !ok || htmlRegex.MatchString(s) {
		return r.FormatError(fmt.Errorf("must not contain HTML tags"))
	}
	return nil
}

func (r *noHTMLRule) Msg(msg string) Rule         { nr := *r; nr.SetMsg(msg); return &nr }
func (r *noHTMLRule) Optional() Rule              { nr := *r; nr.SetOptional(); return &nr }
func (r *noHTMLRule) When(fn func(any) bool) Rule { nr := *r; nr.SetWhen(fn); return &nr }

var NoHTML Rule = &noHTMLRule{}
