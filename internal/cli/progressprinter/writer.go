// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"io"
	"sync"

	"github.com/containerd/console"
	"github.com/moby/term"
)

type Writer interface {
	Start()
	Stop()
	AllEventsDone() (bool, error)
	Event(interface{}) error
	SetProcessing([]string, []string, []func(interface{}) (string, error), string) error
}

// NewWriter returns a new multi-progress writer
func NewWriter(out io.Writer, oneshot bool) (Writer, error) {
	_, isTerminal := term.GetFdInfo(out)
	_, isConsole := out.(console.File)
	// Use TTY writer if console and not printing exactly once
	if isTerminal && isConsole && !oneshot {
		return newTTYWriter(out)
	}
	// Use plain output writer if not TTY or printing exactly once
	return &plainWriter{
		out:      out,
		events:   make(map[string]Event),
		mtx:      &sync.Mutex{},
		hdrPrint: false,
		started:  false,
		oneshot:  oneshot,
	}, nil
}

// Create a new TTY writer for fancy output to TTY
func newTTYWriter(out io.Writer) (Writer, error) {
	return &ttyWriter{
		out:         out,
		events:      make(map[string]Event),
		eventIDs:    make([]string, 0),
		done:        make(chan bool),
		doneWriting: make(chan bool),
		mtx:         &sync.Mutex{},
		first:       true,
	}, nil
}
