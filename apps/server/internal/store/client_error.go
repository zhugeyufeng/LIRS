package store

import "fmt"

type ClientError struct {
	message string
	cause   error
}

func NewClientError(message string) error {
	return &ClientError{message: message}
}

func NewClientErrorf(format string, args ...any) error {
	return NewClientError(fmt.Sprintf(format, args...))
}

func WrapClientError(message string, cause error) error {
	return &ClientError{message: message, cause: cause}
}

func (err *ClientError) Error() string {
	if err == nil {
		return ""
	}
	if err.cause != nil {
		return err.message + ": " + err.cause.Error()
	}
	return err.message
}

func (err *ClientError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func (err *ClientError) ClientMessage() string {
	return err.Error()
}

func clientError(message string) error {
	return NewClientError(message)
}

func clientErrorf(format string, args ...any) error {
	return NewClientErrorf(format, args...)
}
