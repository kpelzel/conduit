// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"path/filepath"
	"strings"
)

// pathParents returns all parents of a specified path
func PathParents(path string) []string {
	parents := []string{}

	parts := strings.Split(filepath.Clean(path), "/")
	parents = append(parents, "/")
	for i := range parts {
		parents = append(parents, strings.Join(parts[:i+1], "/"))
	}

	return parents[:len(parents)-1]
}

// FindParents checks if any path in 'paths' is equal to any of the paths in 'possibleParents'
func FindParents(possibleParents []string, paths []string) []string {
	foundParents := []string{}

	for _, p := range paths {
		for _, pp := range possibleParents {
			if pp == filepath.Clean(p) {
				foundParents = append(foundParents, p)
			}
		}
	}

	return foundParents
}

// FindChildren checks if the targetPath is a prefix to any of the 'paths'
func FindChildren(targetPath string, paths []string) []string {
	foundChildren := []string{}

	for _, p := range paths {
		_, after, found := strings.Cut(filepath.Clean(p), filepath.Clean(targetPath))
		if found {
			b, _, f := strings.Cut(after, "/")
			if f && b == "" {
				foundChildren = append(foundChildren, p)
			}
		}
	}

	return foundChildren
}

// FindExacts checks if the targetPath is exactly the same as any of the 'paths'
func FindExacts(targetPath string, paths []string) []string {
	foundExacts := []string{}

	for _, p := range paths {
		if filepath.Clean(targetPath) == filepath.Clean(p) {
			foundExacts = append(foundExacts, p)
		}
	}

	return foundExacts
}
