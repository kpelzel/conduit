// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Writer for interactive TTYs
type plainWriter struct {
	// Where to ultimately write to
	out io.Writer
	// Event map
	events map[string]Event
	// Event ID list (this is purely for printing in the order that events come in)
	eventIDs []string
	// Mutex for writing
	mtx *sync.Mutex
	// Column headers
	colHeaders []string
	// Maximum widths of columns
	maxColWidths []int
	// Getter function names for transfer details
	getters []string
	// Post-processing functions to print values
	postProcessors []func(interface{}) (string, error)
	// Function name (func() bool)to determine if event is active
	activeFunc string
	// Whether headers have been printed
	hdrPrint bool
	// Whether writer can start writing
	started bool
	// Whether writing exactly once
	oneshot bool
}

// Function to start listening for events to print
func (w *plainWriter) Start() {
	// Lock writer while writing
	w.mtx.Lock()
	defer w.mtx.Unlock()
	// Start writing
	w.started = true
	// Initialize max column widths if not already done
	if w.maxColWidths == nil {
		w.maxColWidths = make([]int, len(w.colHeaders))
		w.calcColWidths()
	}
	// Still print headers if no events
	if len(w.events) == 0 {
		w.print(Event{})
	}
	// Print any events that came before start
	for _, e := range w.eventIDs {
		// Print existing events
		w.print(w.events[e])
	}
}

// Function to stop listening for events to print
func (w *plainWriter) Stop() {
	w.started = false
	// Only print summary if not oneshot writing
	if !w.oneshot {
		// Print summary
		fmt.Fprint(w.out, "\n***\n*** SUMMARY\n***\n\n")
		// Ensure to print headers in summary
		w.hdrPrint = false
		for e := range w.events {
			// Print existing events
			w.print(w.events[e])
		}
	}
}

func (w *plainWriter) AllEventsDone() (bool, error) {
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
//     (see plainWriter.Event()) to get the values from
//  3. Post-processing functions to use for post-processing the data from the
//     getter functions to output strings; these should be supplied in the same
//     order as the getters, since they correspond
func (w *plainWriter) SetProcessing(colHeaders []string, getters []string, postProcessors []func(interface{}) (string, error), activeFunc string) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	w.colHeaders = colHeaders
	w.getters = getters
	w.postProcessors = postProcessors
	w.activeFunc = activeFunc

	return nil
}

// Function to send an Event to the plainWriter for printing
func (w *plainWriter) Event(obj interface{}) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	var exists bool
	var currEvent Event
	var err error
	var eid, isActive interface{}
	var isStr bool
	var isBool bool

	// Create event object to store in the plainWriter
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
		// Append Event to ttyWriter slice of Events
		w.eventIDs = append(w.eventIDs, currEvent.ID)
	}

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
	// Create/Update Event mapping in plainWriter for printer updates
	w.events[currEvent.ID] = currEvent

	// Print event
	if w.started {
		w.print(currEvent)
	}

	return nil
}

// Function to calculate sane column widths if events specified before start
// of printing
func (w *plainWriter) calcColWidths() {
	var currVal interface{}
	var currValStr string
	var err error

	// Get number of rows and columns
	cols := len(w.colHeaders)

	// For each column,
	for c := 0; c < cols; c++ {
		w.maxColWidths[c] = len(w.colHeaders[c])
		// For each event/row,
		for e := range w.events {
			// Call current method
			currVal, err = methodCall(w.events[e].Obj, w.getters[c])
			// Get stringified result
			currValStr, err = w.postProcessors[c](currVal)
			if err != nil {
				fmt.Println("Error in calcColWidths()")
			}
			// Update maximum column cell length
			if len(currValStr) > w.maxColWidths[c] {
				w.maxColWidths[c] = len(currValStr)
			}
		}
	}
}

// Main function to print output
func (w *plainWriter) print(event Event) {
	var currVal interface{}
	var currValStr string
	var err error

	// Get number of rows and columns
	cols := len(w.colHeaders)
	// Storage for all printed cells
	cells := make([]string, cols)

	if event.Obj != nil {
		// For each column,
		for c := 0; c < cols; c++ {
			// Call current method
			currVal, err = methodCall(event.Obj, w.getters[c])
			// Get stringified result
			currValStr, err = w.postProcessors[c](currVal)
			if err != nil {
				fmt.Println("Error in print()")
			}
			// Create current cell with result
			cells[c] = currValStr

			// Update maximum column cell length
			if len(cells[c]) > w.maxColWidths[c] {
				w.maxColWidths[c] = len(cells[c])
			}
		}
	}

	if !w.hdrPrint {
		// Print headers
		w.printLine(w.colHeaders)
		w.hdrPrint = true
	}

	if event.Obj != nil {
		// Print event row
		w.printLine(cells)
	}
}

// Function to print an individual line
func (w *plainWriter) printLine(content []string) {
	var contentStr string
	currStrLen, currPadding := 0, 0
	// For each column,
	for i, s := range content {
		// If not first column, write out tab after previous column
		if i > 0 {
			contentStr += fmt.Sprint(strings.Repeat(" ", tabWidth))
			currStrLen += tabWidth
		}
		// Print column cell
		currPadding = w.maxColWidths[i] - len(s)
		contentStr += fmt.Sprintf("%s%s", s, strings.Repeat(" ", currPadding))
		currStrLen += w.maxColWidths[i]
	}
	fmt.Fprintln(w.out, contentStr)
}
