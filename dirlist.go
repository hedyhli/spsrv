package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func generateDirectoryListing(reqPath, path string, conf *Config) ([]byte, error) {
	dirSort := conf.DirlistSort
	dirReverse := conf.DirlistReverse
	var listing string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return []byte(listing), err
	}
	listing = "# Directory listing\n\n"
	// TODO: custom dirlist header in config
	// Do "up" link first
	reqPath = strings.ReplaceAll(reqPath, "/.", "")
	if reqPath != "/" {
		if strings.HasSuffix(reqPath, "/") {
			reqPath = reqPath[:len(reqPath)-1]
		}
		up := filepath.Dir(reqPath)
		listing += fmt.Sprintf("=> %s %s\n", up, "..")
	}
	// Sort files
	sort.SliceStable(files, func(i, j int) bool {
		if dirReverse {
			i, j = j, i
		}
		if dirSort == "name" {
			return files[i].Name() < files[j].Name()
		} else if dirSort == "size" {
			return files[i].Size() < files[j].Size()
		} else if dirSort == "time" {
			return files[i].ModTime().Before(files[j].ModTime())
		}
		return false // Should not happen
	})
	// Format lines
	for _, file := range files {
		// Skip dotfiles
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}
		// Only list world readable files
		if uint64(file.Mode().Perm())&0444 != 0444 {
			continue
		}
		// Make sure links to directories have a trailing slash,
		// to avoid needless redirects
		relativeUrl := url.PathEscape(file.Name())
		if file.IsDir() {
			relativeUrl += "/"
		}
		listing += fmt.Sprintf("=> %s %s\n", relativeUrl, generatePrettyFileLabel(file, path, conf))
	}
	return []byte(listing), nil
}

func generatePrettyFileLabel(info os.FileInfo, path string, conf *Config) string {
	dirTitles := conf.DirlistTitles
	var size string
	if info.IsDir() {
		size = "        "
	} else if info.Size() < 1024 {
		size = fmt.Sprintf("%4d   B", info.Size())
	} else if info.Size() < (1024 << 10) {
		size = fmt.Sprintf("%4d KiB", info.Size()>>10)
	} else if info.Size() < 1024<<20 {
		size = fmt.Sprintf("%4d MiB", info.Size()>>20)
	} else if info.Size() < 1024<<30 {
		size = fmt.Sprintf("%4d GiB", info.Size()>>30)
	} else if info.Size() < 1024<<40 {
		size = fmt.Sprintf("%4d TiB", info.Size()>>40)
	} else {
		size = "GIGANTIC"
	}

	name := info.Name()
	// TODO: hard coded .gmi file ext
	if dirTitles && filepath.Ext(name) == ".gmi" {
		name = readHeading(path, info)
	}
	if len(name) > 40 {
		name = name[:36] + "..."
	}
	if info.IsDir() {
		name += "/"
	}
	return fmt.Sprintf("%-40s    %s   %v", name, size, info.ModTime().Format("Jan _2 2006"))
}

func readHeading(path string, info os.FileInfo) string {
	filePath := filepath.Join(path, info.Name())
	file, err := os.Open(filePath)
	if err != nil {
		return info.Name()
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[1:])
		}
	}
	return info.Name()
}
