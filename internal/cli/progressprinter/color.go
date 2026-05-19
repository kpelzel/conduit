// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"github.com/morikuni/aec"
)

type colorFunc func(string) string

var (
	nocolor colorFunc = func(s string) string {
		return s
	}

	DoneColor colorFunc = aec.BlueF.Apply
)

func NoColor() {
	DoneColor = nocolor
}
