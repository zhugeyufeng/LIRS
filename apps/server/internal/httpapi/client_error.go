package httpapi

import "fmt"

type clientMessageError struct {
	message string
}

func newClientMessageError(message string) error {
	return clientMessageError{message: message}
}

func newClientMessageErrorf(format string, args ...any) error {
	return newClientMessageError(fmt.Sprintf(format, args...))
}

func (err clientMessageError) Error() string {
	return err.message
}

func (err clientMessageError) ClientMessage() string {
	return err.message
}
