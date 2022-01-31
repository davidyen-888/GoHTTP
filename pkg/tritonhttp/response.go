package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

type Response struct {
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"

	// Header stores all headers to write to the response.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// Response headers:
// Date
// Last-Modified (required only if return type is 200)
// Content-Type (required only if return type is 200)
// Content-Length (required only if return type is 200)
// Connection: close (returned in response to a client “Connection: close” header.)

// Write writes the res to the w.
func (res *Response) Write(w io.Writer) error {
	if err := res.WriteStatusLine(w); err != nil {
		return err
	}
	if err := res.WriteSortedHeaders(w); err != nil {
		return err
	}
	if err := res.WriteBody(w); err != nil {
		return err
	}
	return nil
}

// WriteStatusLine writes the status line of res to w, including the ending "\r\n".
// For example, it could write "HTTP/1.1 200 OK\r\n".
func (res *Response) WriteStatusLine(w io.Writer) error {
	bw := bufio.NewWriter(w)
	statusLine := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, statusText[res.StatusCode])
	_, err := bw.WriteString(statusLine)
	if err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

// WriteSortedHeaders writes the headers of res to w, including the ending "\r\n".
// For example, it could write "Connection: close\r\nDate: foobar\r\n\r\n".
// For HTTP, there is no need to write headers in any particular order.
// TritonHTTP requires to write in sorted order for the ease of testing.
func (res *Response) WriteSortedHeaders(w io.Writer) error {
	bw := bufio.NewWriter(w)
	for k, v := range res.Header {
		headerLine := fmt.Sprintf("%v: %v\r\n", k, v)
		_, err := bw.WriteString(headerLine)
		if err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

// WriteBody writes res' file content as the response body to w.
// It doesn't write anything if there is no file to serve.
func (res *Response) WriteBody(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if res.FilePath == "" {
		return nil
	}
	f, err := os.Open(res.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(bw, f)
	if err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}
