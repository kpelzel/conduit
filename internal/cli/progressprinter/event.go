// Copyright 2026. Triad National Security, LLC. All rights reserved.

package progressprinter

import "time"

type Event struct {
	// Event unique identifier
	ID string
	// Event starting time
	StartTime time.Time
	// Time since start of event
	ElapsedTime time.Duration
	// Event's object
	Obj interface{}
	// Event's spinner
	spinner *spinner
	// Whether event is active or not
	isActive bool
}
