// Copyright 2026. Triad National Security, LLC. All rights reserved.

package ftacmd

import (
	"fmt"

	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/internal/fta/plugin"
)

// parseArgs gets the Action from the provided args
func parseArgs(args []string) (proto.Action, error) {
	if len(args) < 1 {
		return proto.Action_COPY, fmt.Errorf("not enough arguments provided: %v", args)
	}

	aArg := args[0]

	// get action from arguments
	var action proto.Action
	if av, ok := proto.Action_value[aArg]; ok {
		action = proto.Action(av)
	} else {
		return proto.Action_COPY, fmt.Errorf("failed to parse action arg: %v", aArg)
	}

	return action, nil
}

// errToErrs adds the provided error and proto error to a list of FTAPathErrors
func errToErrs(err error, pErr proto.Error) plugin.PluginErrors {
	errs := plugin.PluginErrors{
		Errors: []*plugin.FTAPathError{
			{
				ErrMessage: err,
				PErr:       pErr,
			},
		},
	}

	return errs
}
