package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port             int
	Hostname         string
	RootDir          string
	UserDirEnable    bool
	UserDir          string
	DirlistEnable    bool
	DirlistReverse   bool
	DirlistSort      string
	DirlistTitles    bool
	RestrictHostname string
	CGIPaths         []string
}

var defaultConf = &Config{
	Port:             300,
	Hostname:         "localhost",
	RootDir:          "/var/spartan/",
	DirlistEnable:    true,
	DirlistReverse:   false,
	DirlistSort:      "name",
	DirlistTitles:    true,
	UserDirEnable:    true,
	UserDir:          "public_spartan",
	RestrictHostname: "",
	CGIPaths:         []string{"cgi/"},
}

func LoadConfig(path string) (*Config, error) {
	var err error
	var conf Config
	// Defaults
	conf = *defaultConf

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Println(path, "does not exist, using default configuration values")
		return &conf, nil
	}
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		contents, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		if _, err = toml.Decode(string(contents), &conf); err != nil {
			return nil, err
		}
	}

	// Config validation
	if conf.DirlistSort != "name" && conf.DirlistSort != "time" && conf.DirlistSort != "size" {
		fmt.Println("Warning: DirlistSort config option is not one of name/time/size, defaulting to name.")
		conf.DirlistSort = "name"
	}
	// Strip trailing '/' so /~user to /~user/ redirects can work
	conf.UserDir = strings.TrimRight(conf.UserDir, "/")

	return &conf, nil
}
