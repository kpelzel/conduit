// Copyright 2026. Triad National Security, LLC. All rights reserved.

package util

import (
	"fmt"
	"strconv"
	"strings"

	units "github.com/docker/go-units"
)

// Print a raw byte number from a human-readable string
func ProcessBytes(i interface{}) (int64, error) {
	// Check if interface is string
	s, isString := i.(string)
	if !isString {
		return 0, fmt.Errorf("interface passed to ProcessBytes() is not string")
	}
	// Trim extraneous characters
	s = strings.Trim(s, " \n\r\t")
	// Check if empty interface
	if i == nil || s == "" {
		return 0, nil
	}
	// Convert to bytes from human-readable size
	b, err := units.FromHumanSize(s)
	if err != nil {
		return 0, fmt.Errorf("units.FromHumanSize() failed: %v", err)
	}
	return b, nil
}

// Print a human-readable string from a raw byte number or human-readable
// string
func ProcessNiceBytes(i interface{}) (string, error) {
	// Convert to bytes
	pb, err := ProcessBytes(i)
	if err != nil {
		return "", fmt.Errorf("error converting interface to bytes: %v", err)
	}

	// Convert bytes to string value
	b_str := strconv.Itoa(int(pb))

	// Convert string to integer value
	var b int
	b, err = strconv.Atoi(b_str)
	if err != nil {
		return "", fmt.Errorf("error converting bytes string to int: %v", err)
	}
	// Convert back to human size
	hs := units.HumanSize(float64(b))
	return hs, nil
}
