// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"time"

	"github.com/buger/goterm"
	"github.com/lanl/conduit/internal/cli/processing"
	"github.com/morikuni/aec"
)

// Writer for interactive TTYs
type ttyWriter struct {
	// Where to ultimately write to
	out io.Writer
	// Event map
	events map[string]Event
	// Event ID list
	eventIDs []string
	// Channel for when done monitoring
	done chan bool
	// Channel for when writer is done writing
	doneWriting chan bool
	// Mutex for writing
	mtx *sync.Mutex
	// Timer ticker for updating events in writer
	ticker *time.Ticker
	// Writer starting time
	startTime time.Time
	// Time since start of writer
	elapsedTime time.Duration
	// Width of time when printed
	timeStrWidth int
	// Whether first time printing
	first bool
	// Column headers
	colHeaders []string
	// Getter function names for transfer details
	getters []string
	// Post-processing functions to print values
	postProcessors []func(interface{}) (string, error)
	// Function name (func() bool)to determine if event is active
	activeFunc string
}

const tabWidth = 2

// Function to start listening for events to print
func (w *ttyWriter) Start() {
	go func() {
		// Create ticker for interval of 100ms as update interval
		w.ticker = time.NewTicker(100 * time.Millisecond)
		defer w.Stop()

		// Start time for writer
		w.startTime = time.Now()
		w.elapsedTime = 0

		// Listen for updates
		for {
			select {
			// When done printing
			case <-w.done:
				w.print()
				w.doneWriting <- true
				return
			// When time update
			case <-w.ticker.C:
				w.elapsedTime = time.Since(w.startTime)
				w.timeStrWidth = len(processing.NiceDuration(w.elapsedTime))
				w.print()
			}
		}
	}()
}

// Function to stop listening for events to print
func (w *ttyWriter) Stop() {
	w.done <- true
	// Wait for all writing to be done
	<-w.doneWriting
}

// Function to check if all events are done or not
func (w *ttyWriter) AllEventsDone() (bool, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var result interface{}
	var active bool
	var isBool bool
	var err error
	// For each event,
	for ei := range w.events {
		// Invoke the function to check if still active
		result, err = methodCall(w.events[ei].Obj, w.activeFunc)
		// Return error if couldn't call function
		if err != nil {
			return false, &ErrMethodCall{
				MethodName: w.activeFunc,
				Err:        err,
			}
		}
		// Assert bool type for result
		active, isBool = result.(bool)
		// Return error if couldn't assert bool type
		if !isBool {
			return false, &ErrMethodReturnType{
				MethodName:   w.activeFunc,
				ExpectedType: "bool",
			}
		}
		if active {
			return false, nil
		}
	}
	return true, nil
}

// Function to set what to print:
//  1. Column headers to print at top of output
//  2. Names of getter functions that are methods of the given interface
//     (see ttyWriter.Event()) to get the values from
//  3. Post-processing functions to use for post-processing the data from the
//     getter functions to output strings; these should be supplied in the same
//     order as the getters, since they correspond
func (w *ttyWriter) SetProcessing(colHeaders []string, getters []string, postProcessors []func(interface{}) (string, error), activeFunc string) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.colHeaders = colHeaders
	w.getters = getters
	w.postProcessors = postProcessors
	w.activeFunc = activeFunc

	return nil
}

// Function to send an Event to the ttyWriter for printing
func (w *ttyWriter) Event(obj interface{}) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var exists bool
	var currEvent Event
	var err error
	var eid, isActive interface{}
	var isStr bool
	var isBool bool

	// Create event object to store in the ttyWriter
	currEvent = Event{
		Obj: obj,
	}
	// Get the event ID from first given method
	eid, err = methodCall(obj, w.getters[0])
	if err != nil {
		return &ErrMethodCall{
			MethodName: w.getters[0],
			Err:        err,
		}
	}
	// Ensure output of method is string
	currEvent.ID, isStr = eid.(string)
	if !isStr {
		return &ErrMethodReturnType{
			MethodName:   w.getters[0],
			ExpectedType: "string",
		}
	}
	// Check if Event ID is already tracked by ttyWriter
	exists = false
	for _, i := range w.eventIDs {
		if i == currEvent.ID {
			exists = true
			break
		}
	}
	// Add Event ID if not already tracked
	if !exists {
		// Start tracking time since Event started
		currEvent.StartTime = time.Now()
		currEvent.ElapsedTime = 0
		// Create event spinner
		currEvent.spinner = newSpinner()
		// Append Event to ttyWriter slice of Events
		w.eventIDs = append(w.eventIDs, currEvent.ID)
	} else {
		// If event ID already exists, use existing start time
		currEvent.StartTime = w.events[currEvent.ID].StartTime
		// Update elapsed time
		currEvent.ElapsedTime = time.Since(currEvent.StartTime)
		currEvent.spinner = w.events[currEvent.ID].spinner
	}
	// Create/Update Event mapping in ttyWriter for printer updates
	w.events[currEvent.ID] = currEvent

	// Get the event active status from given function name
	isActive, err = methodCall(obj, w.activeFunc)
	if err != nil {
		return &ErrMethodCall{
			MethodName: w.activeFunc,
			Err:        err,
		}
	}
	// Ensure output of method is string
	currEvent.isActive, isBool = isActive.(bool)
	if !isBool {
		return &ErrMethodReturnType{
			MethodName:   w.activeFunc,
			ExpectedType: "bool",
		}
	}

	return nil
}

// Main function to print output
func (w *ttyWriter) print() {
	// Lock writer while writing
	w.mtx.Lock()
	defer w.mtx.Unlock()

	// Create ANSI builder
	b := aec.EmptyBuilder
	// If first print, don't escape sequence up
	if w.first {
		w.first = false
	} else {
		// If not first print, escape sequence up per event
		for range w.events {
			b = b.Up(1)
		}
		// One more for headers
		b = b.Up(1)
		// Print ANSI sequence for up
		fmt.Fprint(w.out, b.Column(0).ANSI)
	}
	// Hide cursor while printing
	fmt.Fprint(w.out, aec.Hide)
	defer fmt.Fprint(w.out, aec.Show)

	var currVal interface{}
	var currValStr string
	var err error
	var isActive interface{}
	var currActive bool
	// Get number of rows and columns
	rows, cols := len(w.eventIDs), len(w.colHeaders)
	// Storage for all printed cells
	cells := make([][]string, cols)
	// Storage for max column widths (for printing)
	maxColWidths := make([]int, cols)
	var maxLineWidth int

	// For each column,
	for c := 0; c < cols; c++ {
		if cells[c] == nil {
			cells[c] = make([]string, rows)
		}
		maxColWidths[c] = len(w.colHeaders[c])
		// For each event/row,
		for e := 0; e < rows; e++ {
			// Call current method
			currVal, err = methodCall(w.events[w.eventIDs[e]].Obj, w.getters[c])
			// Get stringified result
			currValStr, err = w.postProcessors[c](currVal)
			if err != nil {
			}
			// Create current cell with result
			cells[c][e] = currValStr

			// Update maximum column cell length
			if len(cells[c][e]) > maxColWidths[c] {
				maxColWidths[c] = len(cells[c][e])
			}

			// Check if event is still active and Update event timing on first iteration
			if c == 0 {
				// Get the event active status from given function name
				isActive, _ = methodCall(w.events[w.eventIDs[e]].Obj, w.activeFunc)
				// Ensure output of method is string
				currActive, _ = isActive.(bool)
				// Update if event is active or not
				if event, ok := w.events[w.eventIDs[e]]; ok {
					event.isActive = currActive
					// Stop event's spinner if not active
					if !currActive {
						event.spinner.Stop()
					}
					// Reassign event for printing
					w.events[w.eventIDs[e]] = event
				}

				// Cannot assign struct value directly in map, so we use a copy
				if event, ok := w.events[w.eventIDs[e]]; ok {
					if currActive {
						// Update elapsed time
						event.ElapsedTime = w.elapsedTime
					}
					// Reassign event for printing
					w.events[w.eventIDs[e]] = event
				}
			}
		}
		maxLineWidth += maxColWidths[c]
		// Add minimum tabspace
		if c < (cols - 1) {
			maxLineWidth += tabWidth
		}
	}

	// Get terminal width
	lineWidth := goterm.Width()
	// Get string form of current elapsed time
	currElapsedTime := processing.NiceDuration(w.elapsedTime)
	// Print headers
	w.printLine(strings.Repeat(" ", tabWidth), w.colHeaders, currElapsedTime, maxColWidths, lineWidth, tabWidth, true)

	var currSpinner string
	var currRow []string
	var currEvent Event
	// For each event,
	for r := 0; r < rows; r++ {
		currEvent = w.events[w.eventIDs[r]]
		// Construct row string slice
		currRow = make([]string, cols)
		for c := 0; c < cols; c++ {
			currRow[c] = cells[c][r]
		}
		// Get string form of event's current elapsed time
		currElapsedTime := processing.NiceDuration(w.events[w.eventIDs[r]].ElapsedTime)
		// Get string form of event's spinner
		currSpinner = fmt.Sprintf("%s ", currEvent.spinner.String())
		// Print event row
		w.printLine(currSpinner, currRow, currElapsedTime, maxColWidths, lineWidth, tabWidth, currEvent.isActive)
	}
}

// Function to print an individual line
func (w *ttyWriter) printLine(prefix string, content []string, suffix string, maxColWidths []int, lineWidth int, tabWidth int, isActive bool) {
	var contentStr, lineStr string
	lineStopper := processing.Ellipsis
	currStrLen, currPadding, contentWidth := 0, 0, lineWidth-len(prefix)-len(suffix)
	// For each column,
	for i, s := range content {
		// If not first column, write out tab after previous column
		if i > 0 {
			contentStr += fmt.Sprint(strings.Repeat(" ", tabWidth))
			currStrLen += tabWidth
		}
		// Print column cell
		currPadding = maxColWidths[i] - len(s)
		contentStr += fmt.Sprintf("%s%s", s, strings.Repeat(" ", currPadding))
		currStrLen += maxColWidths[i]
		// If we have overflowed the line length, fix content and stop
		if currStrLen > contentWidth {
			contentStr = contentStr[:contentWidth-len(lineStopper)] + lineStopper
			currStrLen = len(contentStr)
			break
		}
	}
	currPadding = lineWidth - currStrLen - len(suffix) - len(prefix)
	lineStr = fmt.Sprintf("%s%s%s%s\n", prefix, contentStr, strings.Repeat(" ", currPadding), suffix)
	if !isActive {
		lineStr = DoneColor(lineStr)
	}
	fmt.Fprint(w.out, lineStr)
}
