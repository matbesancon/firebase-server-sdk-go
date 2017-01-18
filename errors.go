package firebase

import "fmt"

// ErrValue yields more informative errors by including a generic value
// into the error message
type ErrValue struct {
	val interface{}
	msg string
}

func (e ErrValue) Error() string {
	return fmt.Sprintf("ErrValue: %s - value: %#v", e.msg, e.val)
}
