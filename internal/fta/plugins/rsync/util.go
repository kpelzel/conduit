package rsync

import "bytes"

func ScanLinesWithCR(data []byte, atEOF bool) (int, []byte, error) {
	// Look for either \n or \r
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		advance := i + 1

		if data[i] == '\r' && advance < len(data) && data[advance] == '\n' {
			advance++
		}

		return advance, data[:i], nil
	}

	// If at EOF, return the rest of the data
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}
