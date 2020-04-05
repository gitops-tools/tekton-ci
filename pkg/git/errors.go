package git

import "net/http"

type scmError struct {
	msg    string
	Status int
}

func (s scmError) Error() string {
	return s.msg
}

func IsNotFound(err error) bool {
	e, ok := err.(scmError)
	return ok && e.Status == http.StatusNotFound
}
