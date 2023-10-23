// Package gwerrors contains all common errors used by the gateway.
package errors

import "fmt"

var ErrSessionParse = fmt.Errorf("cannot parse session from context")
var ErrNotFound = fmt.Errorf("the requested resource cannot be found")
var ErrMissingCredentials = fmt.Errorf("the required credentials cannot be found")
