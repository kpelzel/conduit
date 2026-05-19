// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"fmt"
	"reflect"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * FUNCTIONS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Helper function to invoke a method for the given interface, so:
// obj.funcName(params) (interface{}, error)
func methodCall(obj interface{}, funcName string, params ...interface{}) (result interface{}, err error) {
	// Get a reflect.Value representing the function/method
	f := reflect.ValueOf(obj).MethodByName(funcName)
	// Ensure number of parameters matches
	if len(params) != f.Type().NumIn() {
		return nil, fmt.Errorf("Parameter quantity mismatch")
	}
	// Construct parameters
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	// Placeholder for method output
	var res []reflect.Value
	// Invoke method
	res = f.Call(in)
	// Get method result and return it
	result = res[0].Interface()
	return result, nil
}
