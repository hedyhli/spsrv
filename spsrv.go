package main

import (
	"bufio"
	"bytes"
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

type Request struct {
	conn     io.ReadWriteCloser
	netConn  *net.Conn
	vhost    string
	user     string
	path     string // Requested path
	filePath string // Actual file path that does not include the content dir name
	dataLen  int
	data     string
}

const (
	statusSuccess     = 2
	statusRedirect    = 3
	statusClientError = 4
	statusServerError = 5
)

// The following default values are set so that a user would never set any value from the CLI to
// the following. so we can distinguish between user supplied value and the default value.
// The default char is not "" because you can set hostname to "" and it will allow requests to
// any hostname.
// This is not using defaultConf values either because if the config has non-default values, and
// default value is supplied from the CLI, we want to keep taht default value, which is likely what
// user wants.
var cliDefaultChar = ","
var cliDefaultInt = 0

var (
	hostname = flag.StringP("hostname", "h", cliDefaultChar, "Hostname")
	port     = flag.IntP("port", "p", cliDefaultInt, "Port to listen to")
	rootDir  = flag.StringP("dir", "d", cliDefaultChar, "Root content directory")
	confPath = flag.StringP("config", "c", "/etc/spsrv.conf", "Path to config file")
	helpFlag = flag.BoolP("help", "?", false, "Get CLI help")
	versionFlag = flag.BoolP("version", "v", false, "View version and exit")
)

var (
	appVersion = "unknown version"
	buildTime = "date unknown"
	appCommit = "unknown"
)


func main() {
	// Custom usage function because we don't want the "pflag: help requested" message, and
	// we don't want to show the default values.
	flag.Usage = func() {
		fmt.Println(`Usage: spsrv [ [ -c <path> -h <hostname> -p <port> -d <path> ] | --help | --version ]

    -c, --config string     Path to config file
    -d, --dir string        Root content directory
    -h, --hostname string   Hostname
    -p, --port int          Port to listen to`)
	}
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *versionFlag {
		fmt.Printf("spsrv %s, commit %s, built %s", appVersion, appCommit, buildTime)
		return
	}


	conf, err := LoadConfig(*confPath)
	if err != nil {
		fmt.Println("Error loading config")
		fmt.Println(err.Error())
		return
	}

	// This allows users overriding values in config via the CLI
	if *hostname != cliDefaultChar {
		conf.Hostname = *hostname
	}
	if *port != cliDefaultInt {
		conf.Port = *port
	}
	if *rootDir != cliDefaultChar {
		conf.RootDir = *rootDir
	}

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
			log.Println("Error accepting connection:", err.Error())
		}
		log.Println("--> Connection from:", conn.RemoteAddr())
		go handleConnection(conn, conf)
	}
}

// handleConnection handles a request and does the response
func handleConnection(netConn net.Conn, conf *Config) {
	conn := io.ReadWriteCloser(netConn)
	// defer conn.Close()
	defer func() {
		conn.Close()
		log.Println("Closed connection")
	}()

	doneScanningRequest := false
	// Check the size of the request buffer.
	s := bufio.NewScanner(conn)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {

		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if doneScanningRequest {
			// Return a byte
			return 1, data[:1], nil
		}
		// Read request
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, bytes.TrimRight(data[0:i], "\r"), nil
		}
		return 0, nil, nil
	})

	// Sanity check incoming request URL content.
	if ok := s.Scan(); !ok {
		sendResponseHeader(conn, statusClientError, "Request not valid")
		return
	}

	// Parse request
	request := s.Text()
	doneScanningRequest = true
	log.Println("--> Incoming request: \"" + request + "\"")
	host, reqPath, dataLen, err := parseRequest(request)
	if err != nil {
		log.Println("Bad request")
		sendResponseHeader(conn, statusClientError, "Bad request")
		return
	}
	userSubdomainReq := false
	if conf.Hostname != "" {
		if conf.Hostname != host {
			if conf.UserDirEnable && conf.UserSubdomains && strings.HasSuffix(host, conf.Hostname) {
				userSubdomainReq = true
			}
			if !userSubdomainReq {
				log.Println("Request host does not match config value Hostname, returning client error.")
				sendResponseHeader(conn, statusClientError, "No proxying to other hosts!")
				return
			}
		}
	}
	if strings.Contains(reqPath, "..") {
		log.Println("Returning client error (directory traversal)")
		sendResponseHeader(conn, statusClientError, "Stop it with your directory traversal technique!")
		return
	}

	var data string
	if dataLen != 0 {
		log.Println("Reading data, length", dataLen)
		// Read the dataLen amount of data from the data block
		var newData string
		for s.Scan() {
			newData = s.Text()
			if len(data)+len(newData) == dataLen {
				data += newData
				break
			}
			if len(data)+len(newData) > dataLen {
				data += newData[:dataLen-len(data)-1]
			}
			data += newData
		}
	}

	var vhost string
	if userSubdomainReq {
		// TODO: Handle extra dots like a.b.host.name?
		vhost = strings.TrimSuffix(host, "."+conf.Hostname)
	}
	req := &Request{vhost: vhost, path: reqPath, netConn: &netConn, conn: conn, data: data, dataLen: dataLen}

	// Time to fetch the files!
	path := resolvePath(reqPath, conf, req)

	// Check for CGI
	for _, cgiPath := range conf.CGIPaths {
		if strings.HasPrefix(req.filePath, cgiPath) {
			if req.user != "" && (!conf.UserCGIEnable || !conf.UserDirEnable) {
				break
			}
			if req.user != "" && (req.filePath == "" || req.filePath == "/") {
				// TODO: Refactor - ATM `path` would contain the current CGI file wanted
				// But for hitting /~user/, req.filePath is NOT index.gmi
				req.filePath = "index.gmi"
			}
			log.Println("Attempting CGI:", req.filePath)

			ok := handleCGI(conf, req, cgiPath)
			if ok {
				return
			}
			break // CGI failed. just handle the request as if it's a static file.
		}
	}

	// Reaching here means it is a static file
	if dataLen != 0 {
		log.Printf("Got data block of length %v, returning client error.", dataLen)
		sendResponseHeader(conn, statusClientError, "Unwanted input data block received")
		return
	}

	serveFile(conn, reqPath, path, conf)
}

// resolvePath takes in teh request path and returns the cleaned filepath that needs to be fetched.
// It also handles user directories paths /~user/ and /~user if user directories is enabled in the config.
func resolvePath(reqPath string, conf *Config, req *Request) (path string) {
	var user string
	// Handle user subdomains
	if req.vhost != "" {
		user = req.vhost
		path = reqPath
	} else if conf.UserDirEnable && strings.HasPrefix(reqPath, "/~") {
		// Handle tildes
		// Note that user.host.name/~user/ would treat it as a literal folder named /~user/
		// (hence using `else if`)
		bits := strings.Split(reqPath, "/")
		user = bits[1][1:]

		// /~user to /~user/ is somehow able to be handled together with any other /folder to /folder/ redirects
		// So I won't worry about that nor handle it specifically

		req.filePath = strings.TrimPrefix(filepath.Clean(strings.TrimPrefix(reqPath, "/~"+user)), "/")
		path = req.filePath
	}

	if user != "" {
		req.filePath = path
		path = filepath.Join("/home/", user, conf.UserDir, path)
		req.user = user

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
	req.filePath = filepath.Clean(strings.TrimPrefix(path, "/"))
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
					path = strings.TrimSuffix(path, "index.gmi")
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
	if strings.HasSuffix(path, ".gmi") || strings.HasSuffix(path, "/") {
		meta = "text/gemini; lang=en; charset=utf-8" // TODO: configure custom meta string
	}

	log.Println("Serving content:", path)
	sendResponseHeader(conn, statusSuccess, meta)
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
