// Package errors contains all common errors used by the gateway.
package gwerrors

import "fmt"

var ErrSessionParse = fmt.Errorf("cannot parse session from context")
var ErrSessionNotFound = fmt.Errorf("cannot find the session")
var ErrSessionExpired = fmt.Errorf("the session is expired")
var ErrTokenNotFound = fmt.Errorf("the token cannot be found")
var ErrTokenExpired = fmt.Errorf("the token is expired")
var ErrNotFound = fmt.Errorf("the requested resource cannot be found")
var ErrMissingCredentials = fmt.Errorf("the required credentials cannot be found")
var ErrMissingDBResource = fmt.Errorf("the requested resource cannot be found in the DB")
var ErrUnsupportedCLI = fmt.Errorf("you are using an older Renku CLI version that is not supported by this version of Renku")

