// Copyright 2026. Triad National Security, LLC. All rights reserved.

package processing

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	units "github.com/docker/go-units"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * CONSTANTS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// String to print when nothing available
const NoneStr = "<none>"

// Text to display when time has expired
const Expired = "Expired"

// Text to display at end of truncated []string or string
const Ellipsis = " ... "

// Limit on number of characters to print for []string
const StringSliceCharLimit = 40

// Limit on number of characters to print for string
const StringCharLimit = StringSliceCharLimit

// Text to display if object is nil
const Nil = "<null>"

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * FUNCTIONS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Generates a pretty-printed time duration with at most three time units
func NiceDuration(duration time.Duration) string {
	var timeDeltaStr string

	// Check if negative
	isNegative := false
	if duration.Seconds() < 0 {
		isNegative = true
	}
	// Calculate timespan in seconds (rounded down)
	timeDeltaSecs := int64(math.Floor(math.Abs(duration.Seconds())))

	if isNegative {
		if duration.Seconds() > -60 {
			timeDeltaStr = fmt.Sprintf("%.1fs", duration.Seconds())
		} else {
			// Get stringified version
			timeDeltaStr = "-" + NiceSeconds(int(timeDeltaSecs))
		}
	} else {
		if duration.Seconds() < 60 {
			timeDeltaStr = fmt.Sprintf("%.1fs", duration.Seconds())
		} else {
			// Get stringified version
			timeDeltaStr = NiceSeconds(int(timeDeltaSecs))
		}
	}

	return timeDeltaStr
}

// Print a value of seconds in terms of seconds/minutes/hours/days.
func NiceSeconds(input int) (result string) {
	years := math.Floor(float64(input) / 60 / 60 / 24 / 7 / 30 / 12)
	seconds := input % (60 * 60 * 24 * 7 * 30 * 12)
	months := math.Floor(float64(seconds) / 60 / 60 / 24 / 7 / 30)
	seconds = input % (60 * 60 * 24 * 7 * 30)
	weeks := math.Floor(float64(seconds) / 60 / 60 / 24 / 7)
	seconds = input % (60 * 60 * 24 * 7)
	days := math.Floor(float64(seconds) / 60 / 60 / 24)
	seconds = input % (60 * 60 * 24)
	hours := math.Floor(float64(seconds) / 60 / 60)
	seconds = input % (60 * 60)
	minutes := math.Floor(float64(seconds) / 60)
	seconds = input % 60

	if years > 0 {
		result = strconv.Itoa(int(years)) + "y" +
			strconv.Itoa(int(months)) + "M" +
			strconv.Itoa(int(weeks)) + "w" +
			strconv.Itoa(int(days)) + "d" +
			strconv.Itoa(int(hours)) + "h" +
			strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else if months > 0 {
		result = strconv.Itoa(int(months)) + "M" +
			strconv.Itoa(int(weeks)) + "w" +
			strconv.Itoa(int(days)) + "d" +
			strconv.Itoa(int(hours)) + "h" +
			strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else if weeks > 0 {
		result = strconv.Itoa(int(weeks)) + "w" +
			strconv.Itoa(int(days)) + "d" +
			strconv.Itoa(int(hours)) + "h" +
			strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else if days > 0 {
		result = strconv.Itoa(int(days)) + "d" +
			strconv.Itoa(int(hours)) + "h" +
			strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else if hours > 0 {
		result = strconv.Itoa(int(hours)) + "h" +
			strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else if minutes > 0 {
		result = strconv.Itoa(int(minutes)) + "m" +
			strconv.Itoa(int(seconds)) + "s"
	} else {
		result = strconv.Itoa(int(seconds)) + "s"
	}

	return
}

// Print time until event in seconds, hours, etc.
func ProcessTimeUntil(i interface{}) (string, error) {
	obj, found := i.(*timestamppb.Timestamp)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessTimeUntil()")
	}
	// Calculate duration
	timeUntil := time.Until(obj.AsTime()).Round(time.Second)
	return NiceDuration(timeUntil), nil
}

// Print time since event in seconds, hours, etc.
func ProcessTimeSince(i interface{}) (string, error) {
	obj, found := i.(*timestamppb.Timestamp)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessTimeSince()")
	}
	timeSince := time.Since(obj.AsTime()).Round(time.Second)

	return NiceDuration(timeSince), nil
}

// Print time until event in seconds, hours, etc. or Expired
func ProcessTimeToExpire(i interface{}) (string, error) {
	obj, found := i.(*timestamppb.Timestamp)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessTimeToExpire()")
	}
	timeDiff := int32(obj.Seconds) - int32(time.Now().Unix())
	if timeDiff <= 0 {
		return Expired, nil
	}
	return ProcessTimeUntil(i)
}

// Print creation date of timestamp in 'ls -l' style
func ProcessCreationDate(i interface{}) (string, error) {
	obj, found := i.(*timestamppb.Timestamp)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessCreationDate()")
	}
	var createStr string
	nowYear := time.Now().Year()
	createYear := obj.AsTime().Year()
	if nowYear != createYear {
		createStr = obj.AsTime().Local().Format("Jan _2  2006")
	} else {
		createStr = obj.AsTime().Local().Format("Jan _2 15:04")
	}

	return createStr, nil
}

// Print integer as string
func ProcessInt(i interface{}) (string, error) {
	var res string
	switch i.(type) {
	case int:
		res = strconv.FormatInt(int64(i.(int)), 10)
	case uint:
		res = strconv.FormatInt(int64(i.(uint)), 10)
	case int8:
		res = strconv.FormatInt(int64(i.(int8)), 10)
	case uint8:
		res = strconv.FormatInt(int64(i.(uint8)), 10)
	case int16:
		res = strconv.FormatInt(int64(i.(int16)), 10)
	case uint16:
		res = strconv.FormatInt(int64(i.(uint16)), 10)
	case int32:
		res = strconv.FormatInt(int64(i.(int32)), 10)
	case uint32:
		res = strconv.FormatInt(int64(i.(uint32)), 10)
	case int64:
		res = strconv.FormatInt(int64(i.(int64)), 10)
	case uint64:
		res = strconv.FormatInt(int64(i.(uint64)), 10)
	default:
		return "", fmt.Errorf("Invalid type passed to ProcessInt()")
	}
	return res, nil
}

// Print string limited to StringCharLimit characters
func ProcessString(i interface{}) (string, error) {
	str, found := i.(string)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessString()")
	}
	if len(str) > StringCharLimit {
		str = str[:StringCharLimit-len(Ellipsis)] + Ellipsis
	}
	if str == "" {
		str = NoneStr
	}
	return str, nil
}

// Print string slice as 'a, b, c'.
func ProcessStringSlice(i interface{}) (string, error) {
	strList, found := i.([]string)
	if !found {
		return "", fmt.Errorf("Invalid type passed to ProcessStringSlice()")
	}
	var resStr string
	if len(strList) < 1 {
		resStr = NoneStr
	} else if len(strList) == 1 {
		resStr = strList[0]
	} else {
		for sidx, str := range strList {
			if sidx < len(strList)-1 {
				resStr += str + ", "
			} else {
				resStr += str
			}
		}
	}
	if len(resStr) > StringSliceCharLimit {
		resStr = resStr[:StringSliceCharLimit-len(Ellipsis)] + Ellipsis
	}
	return resStr, nil
}

// Print string from STRING to String.
func ProcessCaps(i interface{}) (string, error) {
	strInterface, isStringer := i.(fmt.Stringer)
	s, isString := i.(string)
	if !isStringer && !isString {
		return "", fmt.Errorf("Interface passed to ProcessCaps() does not implement fmt.Stringer")
	}
	var strVal string
	if !isString {
		strVal = strInterface.String()
	} else {
		strVal = s
	}
	strs := strings.Split(strVal, "_")
	var res string
	for i := 0; i < len(strs); i++ {
		strs[i] = strings.ToLower(strs[i])
		currRunes := []rune(strs[i])
		currRunes[0] = unicode.ToUpper(currRunes[0])
		res += string(currRunes)
	}
	if res == "" {
		res = NoneStr
	}
	return res, nil
}

// Print string from PREFIX_STRING to String.
func ProcessCapsSinglePrefix(i interface{}) (string, error) {
	strInterface, isStringer := i.(fmt.Stringer)
	s, isString := i.(string)
	if !isStringer && !isString {
		return "", fmt.Errorf("Interface passed to ProcessCapsSinglePrefix() does not implement fmt.Stringer")
	}
	var strVal string
	if !isString {
		strVal = strInterface.String()
	} else {
		strVal = s
	}
	strs := strings.Split(strVal, "_")
	var res string
	for i := 1; i < len(strs); i++ {
		strs[i] = strings.ToLower(strs[i])
		currRunes := []rune(strs[i])
		currRunes[0] = unicode.ToUpper(currRunes[0])
		res += string(currRunes)
	}
	if res == "" {
		res = NoneStr
	}
	return res, nil
}

// Print a string-string map with limits
func ProcessMapStringString(i interface{}) (string, error) {
	s, isMapStrStr := i.(map[string]string)
	if !isMapStrStr {
		return "", fmt.Errorf("Interface passed to ProcessMapStringString() is not map[string]string")
	}
	res := ""
	map_len := len(s)
	s_idx := 0
	curr_s := ""
	// For each given key/value,
	for s_key, s_val := range s {
		// Determine string based on if at end or not
		if s_idx < map_len-1 {
			curr_s = fmt.Sprintf("%v: %v, ", s_key, s_val)
		} else {
			curr_s = fmt.Sprintf("%v: %v", s_key, s_val)
		}
		// If adding current string would be too long,
		if len(curr_s)+len(res) > StringCharLimit {
			// Fix result and break
			res += curr_s[:StringCharLimit-len(Ellipsis)] + Ellipsis
			break
			// If adding current string is fine,
		} else {
			// Add it
			res += curr_s
		}
		s_idx++
	}
	return res, nil
}

// Print a raw byte number from a human-readable string
func ProcessBytes(i interface{}) (string, error) {
	// Check if interface is string
	s, isString := i.(string)
	if !isString {
		return "", fmt.Errorf("Interface passed to ProcessBytes() is not string")
	}
	// Trim extraneous characters
	s = strings.Trim(s, " \n\r\t")
	// Check if empty interface
	if i == nil || s == "" {
		return "0", nil
	}
	// Convert to bytes from human-readable size
	b, err := units.FromHumanSize(s)
	if err != nil {
		return "", fmt.Errorf("tabbedprinter.ProcessBytes: units.FromHumanSize() failed: %v", err)
	}
	// Convert bytes to string value
	bs := strconv.Itoa(int(b))
	return bs, nil
}

// Print a human-readable string from a rae byte number or human-readable
// string
func ProcessNiceBytes(i interface{}) (string, error) {
	// Convert to bytes
	b_str, err := ProcessBytes(i)
	if err != nil {
		return "", fmt.Errorf("Error converting interface to bytes: %v", err)
	}
	// Convert string to integer value
	var b int
	b, err = strconv.Atoi(b_str)
	if err != nil {
		return "", fmt.Errorf("Error converting bytes string to int: %v", err)
	}
	// Convert back to human size
	hs := units.HumanSize(float64(b))
	return hs, nil
}
