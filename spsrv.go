package main

import (
	"bufio"
	"errors"
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

	flag "github.com/spf13/pflag"
)

const (
	statusSuccess     = 2
	statusRedirect    = 3
	statusClientError = 4
	statusServerError = 5
)

var (
	hostname = flag.StringP("hostname", "h", defaultConf.Hostname, "Hostname")
	port     = flag.IntP("port", "p", defaultConf.Port, "Port to listen to")
	rootDir  = flag.StringP("dir", "d", defaultConf.RootDir, "Root content directory")
	confPath = flag.StringP("config", "c", "/etc/spsrv.conf", "Path to config file")
)

func main() {
	flag.Parse()
	conf, err := LoadConfig(*confPath)
	if err != nil {
		fmt.Println("Error loading config")
		fmt.Println(err.Error())
		return
	}

	// This allows users overriding values in config via the CLI
	if *hostname != defaultConf.Hostname {
		conf.Hostname = *hostname
	}
	if *port != defaultConf.Port {
		conf.Port = *port
	}
	if *rootDir != defaultConf.RootDir {
		conf.RootDir = *rootDir
	}

	// TODO: do something with conf.Hostname (b(like restricting to ipv4/6 etc)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Port))
	if err != nil {
		log.Fatalf("Unable to listen: %s", err)
	}
	log.Println("✨ You are now running on spsrv ✨")
	log.Printf("Listening for connections on port: %d", conf.Port)

	serveSpartan(listener, conf)
}

// serveSpartan accepts connections and returns content
func serveSpartan(listener net.Listener, conf *Config) {
	for {
		// Blocking until request received
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		log.Println("Accepted connection from", conn.RemoteAddr())
		go handleConnection(conn, conf)
	}
}

// handleConnection handles a request and does the response
func handleConnection(conn io.ReadWriteCloser, conf *Config) {
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
	host, reqPath, _, err := parseRequest(request)
	if err != nil {
		sendResponseHeader(conn, statusClientError, "Bad request")
		return
	}
	if conf.RestrictHostname != "" {
		if conf.RestrictHostname != host {
			log.Println("Request host does not match conf.RestrictHostname, returning client error.")
			sendResponseHeader(conn, statusClientError, "No proxying to other hosts!")
			return
		}
	}
	log.Println("Handling request:", request)
	if strings.Contains(reqPath, "..") {
		sendResponseHeader(conn, statusClientError, "Stop it with your directory traversal technique!")
		return
	}

	// Time to fetch the files!
	path := resolvePath(reqPath, conf)
	serveFile(conn, reqPath, path, conf)
	log.Println("Closed connection")
}

func resolvePath(reqPath string, conf *Config) (path string) {
	// Handle tildes
	if conf.UserDirEnable && strings.HasPrefix(reqPath, "/~") {
		bits := strings.Split(reqPath, "/")
		username := bits[1][1:]
		new_prefix := filepath.Join("/home/", username, conf.UserDir)
		path = filepath.Clean(strings.Replace(reqPath, bits[1], new_prefix, 1))
		if strings.HasSuffix(reqPath, "/") {
			path = filepath.Join(path, "index.gmi")
		}
		return
	}
	path = reqPath
	// TODO: [config] default index file for a directory is index.gmi
	if strings.HasSuffix(reqPath, "/") || reqPath == "" {
		path = filepath.Join(reqPath, "index.gmi")
	}
	path = filepath.Clean(filepath.Join(conf.RootDir, path))
	return
}

// serveFile serves opens the requested path and returns the file content
func serveFile(conn io.ReadWriteCloser, reqPath, path string, conf *Config) {
	// If the content directory is not specified as an absolute path, make it absolute.
	// prefixDir := ""
	// var rootDir http.Dir
	// if !strings.HasPrefix(conf.RootDir, "/") {
	// 	prefixDir, _ = os.Getwd()
	// }
	// Avoid directory traversal type attacks.
	// rootDir = http.Dir(prefixDir + strings.Replace(conf.RootDir, ".", "", -1))

	// Open the requested resource.
	var content []byte
	log.Printf("Fetching: %s", path)
	f, err := os.Open(path)
	if err != nil {
		// not putting the /folder to /folder/ redirect here because folder can still
		// be opened without errors
		// Directory listing
		if conf.DirlistEnable && strings.HasSuffix(path, "index.gmi") {
			// fullPath := filepath.Join(fmt.Sprint(rootDir), path)
			fullPath := path
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// If and only if the path is index.gmi AND index.gmi does not exist
				fullPath = strings.TrimSuffix(fullPath, "index.gmi")
				if _, err := os.Stat(fullPath); err == nil {
					// If the directly exists
					log.Println("Generating directory listing:", fullPath)
					content, err = generateDirectoryListing(reqPath, fullPath, conf)
					if err != nil {
						log.Println(err)
						sendResponseHeader(conn, statusServerError, "Error generating directory listing")
						return
					}
					path += ".gmi" // OOF, this is just to have the text/gemini meta later lol
					serveContent(conn, content, path)
					return
				}
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
		if _, err := os.Stat(path + "/"); !os.IsNotExist(err) {
			log.Println("Redirecting", path, "to", reqPath+"/")
			sendResponseHeader(conn, statusRedirect, reqPath+"/")
			return
		}
		log.Println(err)
		sendResponseHeader(conn, statusServerError, "Resource could not be read")
		return
	}
	serveContent(conn, content, path)
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

func parseRequest(r string) (host, path string, contentLength int, err error) {
	parts := strings.Split(r, " ")
	if len(parts) != 3 {
		err = errors.New("Bad request")
		return
	}
	host, path, contentLengthString := parts[0], parts[1], parts[2]
	contentLength, err = strconv.Atoi(contentLengthString)
	if err != nil {
		return
	}
	return
}
