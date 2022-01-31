package tritonhttp

import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// Request headers:
// Host (required, 400 client error if not present)
// Connection (optional, if set to “close” then server should close connection with the client after sending response for this request)
// You should gracefully handle any other valid request headers that the client sends. Any request headers not in the proper form (e.g., missing a colon), should signal a 400 error.

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	// Read start line
	req = &Request{}
	line, err := ReadLine(br)
	if err != nil {
		return nil, false, err
	}
	// Read headers
	req.Header = make(map[string]string)
	for {
		line, err = ReadLine(br)
		if err != nil {
			return nil, true, err
		}
		if line == "" {
			break
		}
		// Parse header
		i := strings.IndexByte(line, ':')
		if i < 0 {
			return nil, true, fmt.Errorf("invalid header: %q", line)
		}
		key := strings.ToLower(line[:i])
		value := strings.TrimSpace(line[i+1:])
		req.Header[key] = value
	}
	// Check required headers

	// Handle special headers

	// Parse the request status line
	req.Method, err = parseRequestLine(line)
	if err != nil {
		return nil, true, err
	}

	// Check for GET HTTP verb
	if req.Method != "GET" {
		return nil, true, fmt.Errorf("invalid method found: %v", req.Method)
	}

	for {
		line, err := ReadLine(br)
		if err != nil {
			return nil, true, err
		}
		if line == "" {
			break
		}
		fmt.Printf("Read line from request: %v", line)
	}

	fmt.Println("Request formed: ", req)
	return req, true, nil
}

func parseRequestLine(line string) (string, error) {
	fields := strings.SplitN(line, " ", 2)
	if len(fields) != 2 {
		return "", fmt.Errorf("could not parse request line, got fields: %v", fields)
	}
	return fields[0], nil
}
