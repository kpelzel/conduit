// Copyright 2026. Triad National Security, LLC. All rights reserved.

package api

import (
	"fmt"
	"strings"
)

// ParseAction will return an Action that closest matches the input string.
// If no matches are found, Action_COPY is returned with an error.
func ParseAction(input string) (Action, error) {
	input = strings.TrimSpace(input)
	switch input {
	case "cp":
		fallthrough
	case "copy":
		fallthrough
	case "COPY":
		fallthrough
	case "CP":
		return Action_COPY, nil
	case "mv":
		fallthrough
	case "move":
		fallthrough
	case "MOVE":
		fallthrough
	case "MV":
		return Action_MOVE, nil
	case "recursive-copy":
		fallthrough
	case "RECURSIVE-COPY":
		fallthrough
	case "recursive-cp":
		fallthrough
	case "RECURSIVE-CP":
		return Action_RECURSIVE_COPY, nil
	case "recursive-move":
		fallthrough
	case "RECURSIVE-MOVE":
		fallthrough
	case "recursive-mv":
		fallthrough
	case "RECURSIVE-MV":
		return Action_RECURSIVE_MOVE, nil
	default:
		return 0, fmt.Errorf("unrecognized action: %v", input)
	}
}
