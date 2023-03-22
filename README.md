# GoHTTP

# GoHTTP Specification

This section describes a minimal subset (which also differs in details) of the HTTP/1.1 protocol specification. Portions of this specification are [courtesy of James Marshall](https://www.jmarshall.com/easy/http/), used with permission from the author.

## Functionality

- A web server listens for connections on a socket bound to a specific port on a host machine
- Clients connect to the socket and use the GoHTTP protocol to retrieve files from the server
- The server reads data from the client and uses framing and parsing techniques to interpret one or more requests (if the client is using pipelined requests)
- Each time the server reads in a full request, it services that request and sends a response back to the client
- After sending back one or more responses, the server will either close the connection if instructed to do so by the client via the “Connection: close” header, or after an appropriate timeout occurs
- The web server should be implemented in a concurrent manner so that it can process multiple client requests overlapping in time.

## Client/Server Protocol

GoHTTP is a client/server protocol that is layered on top of the reliable stream-oriented transport protocol TCP. Clients send request messages to the server, and servers reply with response messages. In its most basic form, a single GoHTTP-level request/response exchange happens over a single, dedicated TCP connection. The client first connects to the server, and then sends the GoHTTP request message. The server replies with a GoHTTP response, and then closes the connection:

Repeatedly setting up and tearing down TCP connections reduces overall network throughput and efficiency, and so GoHTTP has a mechanism whereby a client can reuse a TCP connection to a given server (HTTP **persistent connection**). The idea is that the client opens a TCP connection to the server, issues a GoHTTP request, gets a GoHTTP response, and then issues another GoHTTP request on the already open outbound part of the connection. The server replies with the response, and this can continue through multiple request/response interactions. The client signals the last request by setting a “Connection: close” header, described below. The server indicates that it will not handle additional requests by setting the “Connection: close” header in the response. Note that the client can issue more than one GoHTTP request without necessarily waiting for full HTTP replies to be returned (HTTP **pipelining**).

To support clients that do not properly set the “Connection: close” header, the server must implement a **timeout** mechanism to know when it should close the connection (otherwise it might just wait forever). For this project, you should set a **server timeout of 5 seconds**. If this timeout occurs and the client has sent part of a request, but not a full request, then the server should reply back with a 400 client error (described below). If this timeout occurs and the client has not started sending any part of a new request, the server should simply close the connection.

### HTTP Messages

GoHTTP follows the [general HTTP message format](https://developer.mozilla.org/en-US/docs/Web/HTTP/Messages). And it has some further specifications:

- HTTP version supported: `HTTP/1.1`
- Request method supported: `GET`
- Response status supported:
  - `200 OK`
  - `400 Bad Request`
  - `404 Not Found`
- Request headers:
  - `Host` (required)
  - `Connection` (optional, `Connection: close` has special meaning influencing server logic)
  - Other headers are allowed, but won't have any effect on the server logic
- Response headers:
  - `Date` (required)
  - `Last-Modified` (required for a `200` response)
  - `Content-Type` (required for a `200` response)
  - `Content-Length` (required for a `200` response)
  - `Connection: close` (required in response for a `Connection: close` request, or for a `400` response)
  - Response headers should be written in sorted order for the ease of testing

### Server Logic

When to send a `200` response?

- When a valid request is received, and the requested file can be found.

When to send a `404` response?

- When a valid request is received, and the requested file cannot be found or is not under the doc root.

When to send a `400` response?

- When an invalid request is received.
- When timeout occurs and a partial request is received.

When to close the connection?

- When timeout occurs and no partial request is received.
- When EOF occurs.
- After sending a `400` response.
- After handling a valid request with a `Connection: close` header.

When to update the timeout?

- When trying to read a new request.

What is the timeout value?

- 5 seconds.

## Usage

Install the `httpd` command to a local `bin` directory:

```
make install
ls bin
```

Check the command help message:

```
bin/httpd -h
```

An alternative way to run the command:

```
go run cmd/httpd/main.go -h
```

## Testing

### Sanity Checking

Run an example with the default server:

```
make run-default
```

This example uses the Golang standard library HTTP server to serve the website, and it doesn't rely on your implementation of GoHTTP at all. So you shall be able to run it with the starter code right away. Open the link from output in a browser, and you shall see a test website.

Once you have a working implementation of GoHTTP, you could run another example:

```
make run-gohttp
```

Again, you could use a browser to check the test website served.

### Unit Testing

Unit tests don't involve any networking. They check the logic of the main parts of your implementation.

To run all the unit tests:

```
make unit-test
```

### End-to-End Testing

End-to-end tests involve runing a server locally and testing by communicating with this server.

To run all the end-to-end tests:

```
make e2e-test
```

### Manual Testing

For manual testing, we recommend using `nc`.

In one terminal, start the GoHTTP server:

```
go run cmd/httpd/main.go -port 8080 -doc_root test/testdata/htdocs
```

In another terminal, use `nc` to send request to it:

```
cat test/testdata/requests/single/OKBasic.txt | nc localhost 8080
```

You'll see the response printed out. And you could look at your server's logging to debug.
