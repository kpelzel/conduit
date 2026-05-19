// Copyright 2026. Triad National Security, LLC. All rights reserved.

package fta

import (
	"github.com/lanl/conduit/internal/fta/plugin"
	"github.com/lanl/conduit/internal/fta/plugins/marchive"
	"github.com/lanl/conduit/internal/fta/plugins/pftool"
	"github.com/lanl/conduit/internal/fta/plugins/posix"
	"github.com/lanl/conduit/internal/fta/plugins/rsync"
)

var pluginMap = map[string]plugin.ConduitFTAPlugin{
	// "staging": &staging.StagingPlugin{},
	"rsync":    &rsync.RsyncPlugin{},
	"posix":    &posix.PosixPlugin{},
	"pftool":   &pftool.PftoolPlugin{},
	"marchive": &marchive.MarchivePlugin{},
}
