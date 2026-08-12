// Copyright 2026. Triad National Security, LLC. All rights reserved.

package fta

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	proto "github.com/lanl/conduit/api"
	"github.com/lanl/conduit/defaults"
	"github.com/lanl/conduit/internal/etcd"
	"github.com/lanl/conduit/internal/fta/plugin"
	"github.com/lanl/conduit/internal/logger"
	"github.com/spf13/viper"
)

func StartPluginValidate(log *logger.ConduitLogger, it proto.IncompleteTransfer, em *etcd.ETCDManager, nodeList string) (pluginData *plugin.PluginData, destInfo proto.DestInfo, pluginErrors plugin.PluginErrors) {
	pluginData = &plugin.PluginData{
		SourcePluginInfo:       make(map[string]*plugin.PluginPathInfo),
		DestinationsPluginInfo: make(map[string]*plugin.PluginPathInfo),
		PluginPathData:         make(map[string]*string),
	}

	transferID, err := uuid.Parse(it.GetTransferID())
	if err != nil {
		return pluginData, proto.DestInfo_DEST_NONE, plugin.PluginErrors{
			Errors: []*plugin.FTAPathError{{
				PErr:       proto.Error_ERROR_VALIDATION,
				ErrMessage: fmt.Errorf("failed to parse transfer id[%v]: %v", it.GetTransferID(), err),
			}},
		}
	}

	// get action and options for transfer
	action, options, err := em.GetActionAndOptions(it)
	if err != nil {
		return pluginData, proto.DestInfo_DEST_NONE, plugin.PluginErrors{
			Errors: []*plugin.FTAPathError{{
				PErr:       proto.Error_ERROR_ETCD_CONNECTION,
				ErrMessage: fmt.Errorf("failed to get action and options from etcd: %v", err),
			}},
		}
	}

	// get sources and destination for transfer
	sources, destination, err := em.GetSourcesAndDestination(it)
	if err != nil {
		return pluginData, proto.DestInfo_DEST_NONE, plugin.PluginErrors{
			Errors: []*plugin.FTAPathError{{
				PErr:       proto.Error_ERROR_ETCD_CONNECTION,
				ErrMessage: fmt.Errorf("failed to get source and destination from etcd: %v", err),
			}},
		}
	}

	// glob the sources
	globbedSources := []string{}
	for _, s := range sources {
		gs, err := filepath.Glob(s)
		if err != nil {
			pluginErrors.Warnings = append(pluginErrors.Warnings, &plugin.FTAPathError{
				LeasePath:  s,
				PErr:       proto.Error_ERROR_CONDUIT_INTERNAL,
				ErrMessage: fmt.Errorf("failed to glob source[%v]: %v", s, err),
			})

			globbedSources = append(globbedSources, s)

			continue
		}

		globbedSources = append(globbedSources, gs...)
	}

	// limit character count. This is to prevent a user from passing a wildcard that blows up etcd and slows down queries (it would fail the arg limit at the transfer stage)
	maxSourceBytes := viper.GetInt(defaults.ConfigMaxSourceBytesKey)
	byteCount := 0
	for _, s := range globbedSources {
		byteCount = byteCount + len([]byte(s))
	}

	if byteCount > maxSourceBytes {
		pluginErrors.Errors = []*plugin.FTAPathError{{
			PErr:       proto.Error_ERROR_INVALID_INPUT,
			ErrMessage: fmt.Errorf("request contains too many sources. byte limit: %v, received: %v. Please use a directory instead of a wildcard when transferring a large number of sources", maxSourceBytes, byteCount),
		}}

		return pluginData, proto.DestInfo_DEST_NONE, pluginErrors
	}

	srcPlugins, dstPlugin, pluginErrs := getSrcAndDstValidationPlugins(transferID, log, globbedSources, destination)
	if len(pluginErrs.Errors) > 0 {
		return pluginData, proto.DestInfo_DEST_NONE, pluginErrs
	}

	log.Debugf("sourceplugins: %+v", srcPlugins)

	// add destination to pluginData
	pluginData.DestinationPluginInfo = dstPlugin

	// add sources to pluginData
	for _, s := range globbedSources {
		pluginData.SourcePluginInfo[s] = srcPlugins[s]
	}

	var wg sync.WaitGroup

	var resolvedFTADestinations, userDestinations []string
	var ppd map[string]*string
	var pdLock sync.Mutex
	omitList := []string{}
	var omitLock sync.Mutex

	// get source plugins
	for _, sp := range srcPlugins {
		wg.Add(1)
		go func(goSourcePlugin *plugin.PluginPathInfo) {
			defer wg.Done()
			srcPluginErrors, ppd, omit := sp.Plugin.ValidateSource(goSourcePlugin, action, options)
			pdLock.Lock()
			if omit {
				omitLock.Lock()
				omitList = append(omitList, goSourcePlugin.OriginalUserPath)
				omitLock.Unlock()
			} else {
				pluginData.PluginPathData[goSourcePlugin.OriginalUserPath] = ppd
			}
			pluginErrors.Errors = append(pluginErrors.Errors, srcPluginErrors.Errors...)
			pluginErrors.Warnings = append(pluginErrors.Warnings, srcPluginErrors.Warnings...)
			pdLock.Unlock()
		}(sp)
	}

	wg.Wait()

	for _, o := range omitList {
		delete(pluginData.SourcePluginInfo, o)
	}

	filteredSources := []string{}
	for _, spi := range pluginData.SourcePluginInfo {
		filteredSources = append(filteredSources, spi.OriginalUserPath)
	}

	// check that there are any sources left
	if len(filteredSources) == 0 {
		warnings := ""
		for _, w := range pluginErrors.Warnings {
			if warnings == "" {
				warnings = w.ErrMessage.Error()
			} else {
				warnings = fmt.Sprintf("%v; %v", warnings, w.ErrMessage.Error())
			}
		}

		pluginErrors.Errors = append(pluginErrors.Errors, &plugin.FTAPathError{
			PErr:       proto.Error_ERROR_INVALID_INPUT,
			ErrMessage: fmt.Errorf("No valid sources provided; %v", warnings),
		})

		return pluginData, proto.DestInfo_DEST_NONE, pluginErrors
	}

	var destPluginErrors plugin.PluginErrors
	log.Debugf("sources: %v", globbedSources)
	log.Debugf("destination: %v", destination)
	log.Debugf("dstPlugin.ResolvedFTAPath: %v", dstPlugin.ResolvedFTAPath)
	log.Debugf("dstPlugin.FSC: %v", dstPlugin.FSC)
	destPluginErrors, userDestinations, resolvedFTADestinations, destInfo, ppd = dstPlugin.Plugin.ValidateDestination(filteredSources, destination, dstPlugin.ResolvedFTAPath, dstPlugin.FSC)
	for i, d := range userDestinations {
		pluginData.DestinationsPluginInfo[d] = &plugin.PluginPathInfo{
			OriginalUserPath: destination,
			ResolvedUserPath: d,
			ResolvedFTAPath:  resolvedFTADestinations[i],
			Plugin:           dstPlugin.Plugin,
			FSC:              dstPlugin.FSC,
		}
		pluginData.PluginPathData[d] = ppd[d]
	}
	pluginErrors.Errors = append(pluginErrors.Errors, destPluginErrors.Errors...)
	pluginErrors.Warnings = append(pluginErrors.Warnings, destPluginErrors.Warnings...)

	return pluginData, destInfo, pluginErrors
}
