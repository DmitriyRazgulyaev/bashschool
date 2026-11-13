package domain

import "fmt"

type Object = map[string]any

type Actor struct {
	ID   string
	Role string
}

// Parameters are application filters and identifiers, independent of HTTP.
type Parameters struct {
	ID      string
	Code    string
	Actor   Actor
	Filters map[string]string
}

func (p Parameters) Filter(name, fallback string) string {
	if value, ok := p.Filters[name]; ok {
		return value
	}
	return fallback
}

type Outcome uint8

const (
	Succeeded Outcome = iota
	Created
	NoContent
)

type File struct {
	Data           []byte
	Name           string
	ContentType    string
	StartMS, EndMS *int64
}
type Result struct {
	Value   any
	Outcome Outcome
	File    *File
}

func OK(v any) Result  { return Result{Value: v} }
func New(v any) Result { return Result{Value: v, Outcome: Created} }
func Empty() Result    { return Result{Outcome: NoContent} }
func Download(data []byte, name, contentType string) Result {
	return Result{File: &File{Data: data, Name: name, ContentType: contentType}}
}

type ErrorKind uint8

const (
	Invalid ErrorKind = iota
	Unauthenticated
	Forbidden
	NotFound
	Conflict
	Unavailable
	Internal
)

type Error struct {
	Kind          ErrorKind
	Code, Message string
	Cause         error
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }
func (e *Error) Unwrap() error { return e.Cause }
func Failure(kind ErrorKind, code, message string) error {
	return &Error{Kind: kind, Code: code, Message: message}
}
