package gohttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	responseProto = "HTTP/1.1"

	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	// Validate server configs
	if err := s.ValidateServerSetup(); err != nil {
		return fmt.Errorf("server is not setup correctly %v", err)
	}
	fmt.Println("Server setup valid!")

	// Listen on a port
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	fmt.Println("Listening on", ln.Addr())

	// Accept connections and handle them
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error in accepting connection: %v", err)
			continue
		}
		fmt.Printf("Accepted connection from %v", conn.RemoteAddr())
		go s.HandleConnection(conn)
	}
	// Hint: call HandleConnection
}

func (s *Server) ValidateServerSetup() error {
	fi, err := os.Stat(s.DocRoot)

	if os.IsNotExist(err) {
		return fmt.Errorf("doc_root does not exist: %v", s.DocRoot)
	}

	if !fi.IsDir() {
		return fmt.Errorf("doc_root is not a directory: %v", s.DocRoot)
	}
	return nil
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	fmt.Printf("Handling connection from %v\n", conn.RemoteAddr())
	defer conn.Close()
	br := bufio.NewReader(conn)

	for {
		// Set a read timeout
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
			fmt.Printf("Failed to set timeout for the connection: %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		// Read the next request
		req, bytesReceived, err := ReadRequest(br)

		// Handle errors
		// 1. Client closed connection => io.EOF error
		if errors.Is(err, io.EOF) {
			fmt.Printf("Client closed connection: %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		// 2. Timeout from the server and no partial request is received.=> net.Error error
		// TODO: require more work in proj3
		if err, ok := err.(net.Error); ok && err.Timeout() && req == nil {
			fmt.Printf("Timeout from the server and no partial request is received: %v", conn.RemoteAddr())
			if bytesReceived {
				res := &Response{
					Header: make(map[string]string),
				}
				res.HandleBadRequest()
				res.Write(conn)
			}
			_ = conn.Close()
			return
		}
		// 3. Handle for 400 response, close connection and return
		if err != nil {
			fmt.Printf("Error in reading request: %v", err)
			res := &Response{
				Header: make(map[string]string),
			}
			res.HandleBadRequest()
			res.Write(conn)
			_ = conn.Close()
			return
		}
		// 4. Handle the happy path (200 OK)
		fmt.Printf("Handling good request for %v", req.URL)
		// Handle good request
		res := s.HandleGoodRequest(req)
		fmt.Printf("filepath %s\n", res.FilePath)
		// Write the response
		if err := res.Write(conn); err != nil {
			fmt.Printf("Failed to write response: %v", err)
		}
		// Close conn if requested
		if req.Close {
			_ = conn.Close()
		}
	}
	// Hint: use the other methods below
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	res = &Response{
		Header: make(map[string]string),
	}
	res.Proto = responseProto
	res.StatusCode = statusOK
	url := filepath.Clean(req.URL)
	res.FilePath = path.Join(s.DocRoot, url) // TODO: handle path
	// Hint: use the other methods below

	// Handle for 404 response (a valid request is received, and the requested file cannot be found or is not under the doc root.)
	// Check if file exist
	path, err := os.Stat(res.FilePath)
	if err != nil {
		fmt.Printf("Error in checking if file exists: %v\n", err)
		fmt.Printf("path: %v\n", path)
		res.FilePath = ""
		res.HandleNotFound(req)
		return res
		// Check if it's a folder, if so with /, add index.html, if not , return file not found
	} else if path.IsDir() {
		fmt.Printf("File is a directory: %v", res.FilePath)
		if strings.HasSuffix(url, "/") {
			res.FilePath = filepath.Join(res.FilePath, "index.html")
		} else {
			// file not found
			res.FilePath = ""
			res.HandleNotFound(req)
			return res
		}
	}

	// Check if file is outside root
	if !strings.HasPrefix(res.FilePath, s.DocRoot) {
		fmt.Printf("File is outside root: %v", res.FilePath)
		res.FilePath = ""
		res.HandleNotFound(req)
		return res
	}
	// HandleOk
	res.HandleOK(req, res.FilePath)
	return res
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	stat, err := os.Stat(path)
	res.Header["Date"] = FormatTime((time.Now()))
	res.Header["Last-Modified"] = FormatTime(stat.ModTime())
	res.Header["Content-Type"] = MIMETypeByExtension(filepath.Ext(path))
	res.Header["Content-Length"] = strconv.Itoa(int(stat.Size()))
	if req.Close {
		res.Header["Connection"] = "close"
	}
	if err != nil {
		res.StatusCode = statusNotFound
	}
	res.Proto = responseProto
	res.StatusCode = statusOK
	res.FilePath = path
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {
	res.Header["Date"] = FormatTime((time.Now()))
	res.Proto = responseProto
	res.StatusCode = statusBadRequest
	res.FilePath = ""
	res.Header["Connection"] = "close"
}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	res.Header["Date"] = FormatTime((time.Now()))
	res.Proto = responseProto
	res.StatusCode = statusNotFound
	if req.Close {
		res.Header["Connection"] = "close"
	}
}
