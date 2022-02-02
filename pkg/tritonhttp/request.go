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
	req = &Request{
		Header: make(map[string]string),
	}

	// Read start line
	line, err := ReadLine(br)

	if err != nil {
		return nil, false, err
	}
	// Parse the request status line
	req.Method, req.URL, req.Proto, req.Host, err = parseRequestLine(line)
	if err != nil {
		return nil, true, err
	}
	// Check for GET HTTP verb
	if req.Method != "GET" {
		return nil, true, fmt.Errorf("invalid method found: %v", req.Method)
	}

	// url should start with '/'
	if req.URL[0] != '/' {
		return nil, true, fmt.Errorf("invalid url found: %v", req.URL)
	}

	// protocol should be HTTP/1.1
	if req.Proto != "HTTP/1.1" {
		return nil, true, fmt.Errorf("invalid protocol found: %v", req.Proto)
	}

	// Read headers
	req.Header = make(map[string]string)
	for {
		line, err := ReadLine(br)
		// fmt.Print("line: ", line, "\n")
		if line == "" {
			break
		}
		if err != nil {
			fmt.Print(err.Error())
			return req, true, err
		}

		spilted := strings.Split(line, ":")
		// seperate the header key and value
		// key should not have spaces, only alphanumeric characters and numbers
		IsLetter := func(r rune) bool {
			return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		}
		IsNumber := func(r rune) bool {
			return (r >= '0' && r <= '9')
		}
		IsAlphanumeric := func(r rune) bool {
			return IsLetter(r) || IsNumber(r) || r == '-'
		}
		// check if the key is valid
		key := spilted[0]

		if strings.IndexFunc(key, IsAlphanumeric) != -1 {
			// check if the value is valid
			value := strings.TrimLeft(spilted[1], " ")
			key = CanonicalHeaderKey(key)
			if key == "Host" {
				req.Host = value
			} else if key == "Connection" {
				req.Close = value == "close"
			} else {
				req.Header[key] = value
			}
		} else {
			return nil, true, fmt.Errorf("invalid header key found: %v", key)
		}

	}
	return req, true, nil
}

func parseRequestLine(line string) (string, string, string, string, error) {
	fields := strings.SplitN(line, " ", 3)
	if len(fields) != 3 {
		return "", "", "", "", fmt.Errorf("invalid request line: %v", line)
	}
	return fields[0], fields[1], fields[2], "", nil
}
