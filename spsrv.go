package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	statusSuccess          = 2
	statusRedirectTemp     = 3
	statusClientError = 4
	statusServerError = 5
)

var (
	hostname   = flag.String("h", "localhost", "hostname")
	contentDir = flag.String("d", "/var/gemini", "content directory")
	port       = flag.Int("p", 300, "port number")
)

func main() {
	flag.Parse()

	// Create TSL over TCP session.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Unable to listen: %s", err)
	}
	log.Printf("Listening for connections on port: %d", *port)

	serveGemini(listener)
}

func serveGemini(listener net.Listener) {
	// serve forever
	for {
		// Accept incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		log.Println("Accepted connection")

		go handleConnection(conn)
	}
}

func handleConnection(conn io.ReadWriteCloser) {
	defer conn.Close()

	// Check the size of the request buffer.
	s := bufio.NewScanner(conn)
	if len(s.Bytes()) > 1024 {
		sendResponseHeader(conn, statusClientError, "Request exceeds maximum permitted length")
		return
	}

	// Sanity check incoming request URL content.
	if ok := s.Scan(); !ok {
		sendResponseHeader(conn, statusClientError, "Request not valid")
		return
	}

	// Parse incoming request URL.
	request := s.Text()
	path, _, err := parseRequest(request)
	if err != nil {
		sendResponseHeader(conn, statusClientError, "Bad request")
		return
	}
	log.Println("Handling request:", request)

	// Time to fetch the files!
	// If the URL ends with a '/' character, assume that the user wants the index.gmi
	// file in the corresponding directory.
	var reqPath string
	if strings.HasSuffix(path, "/") || path == "" {
		reqPath = filepath.Join(path, "index.gmi")
	} else {
		reqPath = path
	}
	cleanPath := filepath.Clean(reqPath)

	// If the content directory is not specified as an absolute path, make it absolute.
	var workDir string
	var rootDir http.Dir
	if !strings.HasPrefix(*contentDir, "/") {
		workDir, _ = os.Getwd()
		// Use this function to avoid directory traversal type attacks.
		rootDir = http.Dir(workDir + strings.Replace(*contentDir, ".", "", -1))
	} else {
		rootDir = http.Dir(strings.Replace(*contentDir, ".", "", -1))
	}

	// Open the requested resource.
	log.Printf("Path: %s", cleanPath)
	f, err := rootDir.Open(cleanPath)
	if err != nil {
		// Guess what?? there's an echo handler for this static file server!!
		// Lol cause why not :)
		// once I add configs I'll have an option to disable this
		// this only works if there isn't a file named "echo" in the directory
		// (still wondering where in the world I should put the contentLength check)
		// if contentLength > 0 {
		// 	if ok := s.Scan(); !ok {
		// 		sendResponseHeader(conn, statusPermanentFailure, "Unable to read input content")
		// 		return
		// 	}
		// 	log.Println("Handling /echo request with content length", contentLength)
		// 	content := s.Text()
		// 	echoFunction(conn, content)
		// 	return
		// }
		log.Println(err)
		sendResponseHeader(conn, statusClientError, "Resource not found")
		return
	}
	defer f.Close()

	// Read the contents of the file.
	content, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println(err)
		sendResponseHeader(conn, statusServerError, "Resource could not be read")
		return
	}

	// Determine MIME type.
	meta := http.DetectContentType(content)
	if strings.HasSuffix(cleanPath, ".gmi") {
		meta = "text/gemini; lang=en; charset=utf-8"
	}

	log.Println("Writing response header")
	sendResponseHeader(conn, statusSuccess, meta)

	log.Println("Writing content")
	sendResponseContent(conn, content)

	log.Println("Closed connection")

}

func echoFunction(conn io.ReadWriteCloser, content string) {
	sendResponseHeader(conn, statusSuccess, "text/plain")
	sendResponseContent(conn, []byte(content))
}

func sendResponseHeader(conn io.ReadWriteCloser, statusCode int, meta string) {
	header := fmt.Sprintf("%d %s\r\n", statusCode, meta)
	_, err := conn.Write([]byte(header))
	if err != nil {
		log.Printf("There was an error writing to the connection: %s", err)
	}
}

func sendResponseContent(conn io.ReadWriteCloser, content []byte) {
	_, err := conn.Write(content)
	if err != nil {
		log.Printf("There was an error writing to the connection: %s", err)
	}
}

func parseRequest(r string) (path string, contentLength int, err error) {
	parts := strings.Split(r, " ")
	if len(parts) != 3 {
		err = errors.New("Bad request")
		return
	}
	_, path, contentLengthString := parts[0], parts[1], parts[2]
	contentLength, err = strconv.Atoi(contentLengthString)
	if err != nil {
		return
	}
	return
}
