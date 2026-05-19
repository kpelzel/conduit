// Copyright 2026. Triad National Security, LLC. All rights reserved.

package posix

import (
	"fmt"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/internal/fta/plugin"
)

func (p *PosixPlugin) Setup(transferID uuid.UUID, pathInfo *plugin.PluginPathInfo, pathType proto.LeaseType, action proto.Action, baseDest bool, updateTransferProgress plugin.UpdateTransferProgress) (plugin.PluginErrors, *plugin.PluginPathInfo) {

	// pathInfo.TransferPath tells the transfer plugin what final path to use for its transfer
	pathInfo.TransferPath = pathInfo.ResolvedFTAPath

	updateTransferProgress(proto.ETCDStatusDetails{
		PluginStatus: fmt.Sprintf("setup %v complete", pathInfo.OriginalUserPath),
	})

	return plugin.PluginErrors{}, pathInfo
}
