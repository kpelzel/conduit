// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"fmt"
)

type ErrMethodCall struct {
	MethodName string
	Err        error
}

func (e *ErrMethodCall) Error() string {
	return fmt.Sprintf("Error calling method '%s()' on object: '%v'", e.MethodName, e.Err)
}

type ErrMethodReturnType struct {
	MethodName   string
	ExpectedType string
}

func (e *ErrMethodReturnType) Error() string {
	return fmt.Sprintf("Value returned from method '%s()' not a %s", e.MethodName, e.ExpectedType)
}
