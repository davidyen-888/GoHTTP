package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
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
	br := bufio.NewReader(conn)

	for {
		// Set a read timeout
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
			fmt.Printf("Failed to set timeout for the connection: %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		// Read the next request
		req, _, err := ReadRequest(br)

		// Handle errors
		// 1. Client closed connection => io.EOF error
		if errors.Is(err, io.EOF) {
			fmt.Printf("Client closed connection: %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		// 2. Timeout from the server => net.Error error
		if err, ok := err.(net.Error); ok && err.Timeout() {
			fmt.Printf("Connection to %v timed out", conn.RemoteAddr())
			_ = conn.Close()
			return
		}
		// 3. Malformed/invalid request => error
		// Handle the request which is not a GET and immediately close the connection and return
		if err != nil {
			fmt.Printf("Handling bad request for error: %v", err)
			res := &Response{}
			res.HandleBadRequest()
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}
		// 4. Handle the happy path
		fmt.Printf("Handling good request for %v", req.URL)
		// Handle good request
		res := s.HandleGoodRequest(req)
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
	res = &Response{}
	res.Proto = responseProto
	res.StatusCode = statusOK
	res.FilePath = path.Join(s.DocRoot, req.URL) // TODO: handle path
	return res
	// Hint: use the other methods below
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	res.Proto = responseProto
	res.StatusCode = statusOK
	res.FilePath = path
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client.
func (res *Response) HandleBadRequest() {
	res.Proto = responseProto
	res.StatusCode = statusBadRequest
	res.FilePath = "" // TODO: handle path
}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	res.Proto = responseProto
	res.StatusCode = statusNotFound
	res.FilePath = "" // TODO: handle path
}
