// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"strings"

	proto "github.com/lanl/conduit/api"
)

func CleanKrbCache(input string) (cacheType proto.KrbCacheType, cachePath string, err error) {
	cacheType = proto.KrbCacheType_KRB_NONE
	cachePath = input

	// check if input has a prefix of a cache type
	for kType, kInt := range proto.KrbCacheType_value {
		prefix := fmt.Sprintf("%s:", kType)
		if strings.HasPrefix(input, prefix) {
			// set the cacheType for return
			cacheType = proto.KrbCacheType(kInt)
			// trim the cache prefix
			cachePath = strings.TrimPrefix(input, prefix)
			break
		}
	}

	return cacheType, cachePath, nil
}
