package apperror

type Kind string

const (
	KindUnauthenticated Kind = "unauthenticated"
	KindForbidden       Kind = "forbidden"
	KindNotFound        Kind = "not_found"
	KindInvalid         Kind = "invalid"
	KindConflict        Kind = "conflict"
)

// Detail provides structured context about an error without coupling it to a
// particular transport such as HTTP or gRPC.
type Detail struct {
	Field string
	Value any
}

// Coded is implemented by errors that are safe to expose to clients.
type Coded interface {
	error
	Kind() Kind
	Code() string
	Message() string
	Details() []Detail
}

type Error struct {
	kind    Kind
	code    string
	message string
	details []Detail
}

func New(kind Kind, code, message string, details ...Detail) *Error {
	return &Error{
		kind:    kind,
		code:    code,
		message: message,
		details: append([]Detail(nil), details...),
	}
}

func (e *Error) Kind() Kind {
	return e.kind
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Code() string {
	return e.code
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Details() []Detail {
	return append([]Detail(nil), e.details...)
}
