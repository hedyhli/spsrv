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
	statusSuccess     = 2
	statusRedirect    = 3
	statusClientError = 4
	statusServerError = 5
)

var (
	hostname   = flag.String("h", "localhost", "hostname")
	contentDir = flag.String("d", "/var/spartan", "content directory")
	port       = flag.Int("p", 300, "port number")
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Unable to listen: %s", err)
	}
	log.Printf("Listening for connections on port: %d", *port)

	serveSpartan(listener)
}

// serveSpartan accepts connections and returns content
func serveSpartan(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		log.Println("Accepted connection")
		go handleConnection(conn)
	}
}

// handleConnection handles a request and does the reponse
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
	serveFile(conn, path)
	log.Println("Closed connection")
}

// serveFile serves opens the requested path and returns the file content
func serveFile(conn io.ReadWriteCloser, reqPath string) {
	// TODO: [config] default index file for a directory is index.gmi
	path := reqPath
	if strings.HasSuffix(reqPath, "/") || reqPath == "" {
		path = filepath.Join(reqPath, "index.gmi")
	}
	cleanPath := filepath.Clean(path)

	// If the content directory is not specified as an absolute path, make it absolute.
	prefixDir := ""
	var rootDir http.Dir
	if !strings.HasPrefix(*contentDir, "/") {
		prefixDir, _ = os.Getwd()
	}
	// Avoid directory traversal type attacks.
	rootDir = http.Dir(prefixDir + strings.Replace(*contentDir, ".", "", -1))

	// Open the requested resource.
	var content []byte
	log.Printf("Fetching: %s", cleanPath)
	f, err := rootDir.Open(cleanPath)
	if err != nil {
		// not putting the /folder to /folder/ redirect here because folder can still
		// be opened without errors
		// Directory listing
		if strings.HasSuffix(cleanPath, "index.gmi") {
			fullPath := filepath.Join(fmt.Sprint(rootDir), cleanPath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// If and only if the path is index.gmi AND index.gmi does not exist
				fullPath = strings.TrimSuffix(fullPath, "index.gmi")
				log.Println("Generating directory listing:", fullPath)
				content, err = generateDirectoryListing(reqPath, fullPath)
				if err != nil {
					log.Println(err)
					sendResponseHeader(conn, statusServerError, "Error generating directory listing")
					return
				}
				cleanPath += ".gmi" // OOF, this is just to have the text/gemini meta later lol
				serveContent(conn, content, cleanPath)
				return
			}
		}
		log.Println(err)
		sendResponseHeader(conn, statusClientError, "Not found")
		return
	}
	defer f.Close()

	// Read da file
	content, err = ioutil.ReadAll(f)
	if err != nil {
		// /folder to /folder/ redirect
		// I wish I could check if err is a "path/to/dir" is a directory error
		// but I couldn't figure out how, so this check below is the best I
		// can come up with I guess
		if _, err := os.Stat(filepath.Join(fmt.Sprint(rootDir), cleanPath+"/")); !os.IsNotExist(err) {
			log.Println("Redirecting", cleanPath, "to", cleanPath+"/")
			sendResponseHeader(conn, statusRedirect, cleanPath+"/")
			return
		}
		log.Println(err)
		sendResponseHeader(conn, statusServerError, "Resource could not be read")
		return
	}
	serveContent(conn, content, cleanPath)
}

func serveContent(conn io.ReadWriteCloser, content []byte, path string) {
	// MIME
	meta := http.DetectContentType(content)
	if strings.HasSuffix(path, ".gmi") {
		meta = "text/gemini; lang=en; charset=utf-8" // TODO: configure custom meta string
	}

	log.Println("Writing response header")
	sendResponseHeader(conn, statusSuccess, meta)
	log.Println("Writing content")
	sendResponseContent(conn, content)

}

// func echoFunction(conn io.ReadWriteCloser, content string) {
// 	sendResponseHeader(conn, statusSuccess, "text/plain")
// 	sendResponseContent(conn, []byte(content))
// }

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
