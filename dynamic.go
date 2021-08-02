package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func handleCGI(conf *Config, req *Request, cgiPath string) (ok bool) {
	ok = true
	path := req.filePath
	conn := req.conn
	scriptPath := filepath.Join(conf.RootDir, req.filePath)

	if req.user != "" {
		scriptPath = filepath.Join("/home", req.user, conf.UserDir, req.filePath)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		ok = false
		return
	}
	if !(info.Mode().Perm()&0555 == 0555) {
		ok = false
		return
	}

	// Prepare environment variables
	vars := prepareCGIVariables(conf, req, scriptPath)

	log.Println("Running script:", scriptPath)

	// Spawn process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, scriptPath)

	// Put input data into stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Error creating a stdin pipe:", err.Error())
		ok = false
		return
	}
	io.WriteString(stdin, req.data)
	stdin.Close()

	// Set environment variables
	cmd.Env = []string{}
	for key, value := range vars {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	// Manually change the uid/gid for the command
	// Fetch user info
	// user, err := user.Lookup(req.user)
	// if err == nil {
	// 	tmp, _ := strconv.ParseUint(user.Uid, 10, 32)
	// 	uid := uint32(tmp)
	// 	tmp, _ = strconv.ParseUint(user.Gid, 10, 32)
	// 	gid := uint32(tmp)
	// 	cmd.SysProcAttr = &syscall.SysProcAttr{}
	// 	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}
	// }

	// Fetch and check output
	response, err := cmd.Output()

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Terminating CGI process " + path + " due to exceeding 10 second runtime limit.")
		conn.Write([]byte("42 CGI process timed out!\r\n"))
		return
	}
	if err != nil {
		log.Println("Error running CGI program " + path + ": " + err.Error())
		if err, ok := err.(*exec.ExitError); ok {
			log.Println("↳ stderr output: " + string(err.Stderr))
		}
		conn.Write([]byte("42 CGI error\r\n"))
		return
	}
	// Extract response header
	header, _, err := bufio.NewReader(strings.NewReader(string(response))).ReadLine()
	_, err2 := strconv.Atoi(strings.Fields(string(header))[0])
	if err != nil || err2 != nil {
		log.Println("Unable to parse first line of output from CGI process " + path + " as valid Gemini response header.  Line was: " + string(header))
		conn.Write([]byte("42 CGI error\r\n"))
		return
	}
	log.Println("Returning CGI output")
	// Write response
	conn.Write(response)
	return
}

func prepareCGIVariables(conf *Config, req *Request, script_path string) map[string]string {
	vars := prepareGatewayVariables(conf, req)
	vars["GATEWAY_INTERFACE"] = "CGI/1.1"
	vars["SCRIPT_PATH"] = script_path
	return vars
}

func prepareGatewayVariables(conf *Config, req *Request) map[string]string {
	vars := make(map[string]string)
	// vars["QUERY_STRING"] = URL.RawQuery
	vars["REQUEST_METHOD"] = ""
	vars["SERVER_NAME"] = conf.Hostname
	vars["SERVER_PORT"] = strconv.Itoa(conf.Port)
	vars["SERVER_PROTOCOL"] = "SPARTAN"
	vars["SERVER_SOFTWARE"] = "SPSRV"

	host, _, _ := net.SplitHostPort((*req.netConn).RemoteAddr().String())
	vars["REMOTE_ADDR"] = host
	return vars
}
