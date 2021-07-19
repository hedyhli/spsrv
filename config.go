package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
)

type Config struct {
	Port          int
	Hostname      string
	RootDir       string
	UserDirEnable bool
	UserDir       string
	// UserSlug string
	DirlistReverse bool
	DirlistSort    string
	DirlistTitles  bool
}

var defaultConf = &Config{
	Port:           300,
	Hostname:       "localhost",
	RootDir:        "/var/spartan/",
	DirlistReverse: false,
	DirlistSort:    "name",
	DirlistTitles:  true,
	UserDirEnable:  false,
	UserDir:        "public_spartan",
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

	if conf.DirlistSort != "name" && conf.DirlistSort != "time" && conf.DirlistSort != "size" {
		fmt.Println("Warning: DirlistSort config option is not one of name/time/size, defaulting to name.")
		conf.DirlistSort = "name"
	}

	return &conf, nil
}
