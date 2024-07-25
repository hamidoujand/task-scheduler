// Package errs is used for sending a unified error response back to client and
// also contains some general purposed error types.
package errs

import (
	"fmt"
	"net/http"
	"runtime"
)

// AppError represents a trusted error inside the system
type AppError struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	FuncName string            `json:"-"`
	FileName string            `json:"-"`
	Fields   map[string]string `json:"fields,omitempty"`
}

func (err *AppError) Error() string {
	return err.Message
}

// NewAppError creates an *AppError directly from another error
func NewAppError(code int, message string) error {
	//skip one call stack
	pc, filename, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()

	return &AppError{
		Code:     code,
		Message:  message,
		FuncName: funcName,
		FileName: fmt.Sprintf("%s:%d", filename, line),
	}
}

// NewAppError creates a *AppError with formatted message.
func NewAppErrorf(code int, format string, v ...any) error {
	pc, filename, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return &AppError{
		Code:     code,
		Message:  fmt.Sprintf(format, v...),
		FuncName: funcName,
		FileName: fmt.Sprintf("%s:%d", filename, line),
	}
}

// NewAppValidationError returns an error from failed fields.
func NewAppValidationError(code int, message string, fields map[string]string) error {
	pc, filename, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return &AppError{
		Code:     code,
		Message:  message,
		FuncName: funcName,
		FileName: fmt.Sprintf("%s:%d", filename, line),
		Fields:   fields,
	}
}

// NewAppInternalErr is used to make returning Internal Server Errors easier.
func NewAppInternalErr(err error) error {
	pc, filename, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return &AppError{
		Code:     http.StatusInternalServerError,
		Message:  err.Error(),
		FuncName: funcName,
		FileName: fmt.Sprintf("%s:%d", filename, line),
	}
}
