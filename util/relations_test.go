// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"testing"
)

const (
	testLeasePath = "/foo/bar/hello"
)

var (
	testChildren = []string{
		"/foo/bar/hello/hello",
		"/foo/bar/hello/foo",
		"/foo/bar/hello/hello/foo",
		"/foo/bar/hello/blah",
		"/foo/bar/hello/hello/",
		"/foo/bar/hello/foo/",
		"/foo/bar/hello/hello/foo/",
		"/foo/bar/hello/blah/",
	}
	testSiblings = []string{
		"/foo/bar/goodbye",
		"/foo/bar/blah",
		"/foo/bar/goodbye/",
		"/foo/bar/blah/",
		"/foo/bar/hello-foo",
		"/foo/bar/hello-foo/",
	}
	testParents = []string{
		"/foo/bar",
		"/foo",
		"/foo/bar/",
		"/foo/",
	}
	testCousins = []string{
		"/foo/bar/hello-foo/blah",
	}
)

func concatMultipleSlices(slices [][]string) []string {
	var totalLen int

	for _, s := range slices {
		totalLen += len(s)
	}

	result := make([]string, totalLen)

	var i int

	for _, s := range slices {
		i += copy(result[i:], s)
	}

	return result
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func TestFindChildren(t *testing.T) {
	testset := concatMultipleSlices([][]string{testChildren, testSiblings, testParents, testCousins, {testLeasePath}})

	r := FindChildren(testLeasePath, testset)
	if len(r) < len(testChildren) {
		t.Fatalf("failed to find all children\ndidn't find: %v", difference(testChildren, r))
	}
	if len(r) > len(testChildren) {
		t.Fatalf("found too many children\nfound: %v", difference(r, testChildren))
	}
}

func TestFindParents(t *testing.T) {
	testset := concatMultipleSlices([][]string{testChildren, testSiblings, testParents, testCousins, {testLeasePath}})

	possibleParents := PathParents(testLeasePath)
	r := FindParents(possibleParents, testset)
	if len(r) < len(testParents) {
		t.Fatalf("failed to find all parents\ndidn't find: %v", difference(testParents, r))
	}
	if len(r) > len(testParents) {
		t.Fatalf("found too many parents\nfound: %v", difference(r, testParents))
	}
}
func TestFindExacts(t *testing.T) {
	testset := concatMultipleSlices([][]string{testChildren, testSiblings, testParents, testCousins, {testLeasePath}})

	r := FindExacts(testLeasePath, testset)
	if len(r) < 1 {
		t.Fatalf("failed to find all exacts\ndidn't find: %v", difference([]string{testLeasePath}, r))
	}
	if len(r) > 1 {
		t.Fatalf("found too many exacts\nfound: %v", difference(r, []string{testLeasePath}))
	}
	if r[0] != testLeasePath {
		t.Fatalf("found incorrect exact: %v", r[0])
	}
}
