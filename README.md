# spsrv

A static spartan server with many features:

* folder redirects
* /~user directories
* directory listing
* CONF or TOML config file
  * directory listing options
  * user directory feature and userdir path
* CGI
  * per user CGI (unsafe, like molly-brown)
  * input data as stdin pipe

## install

### with `go get`
first, you need to have go installed and have a folder `~/go` with `$GOPATH` pointing to it.
then you can call `go get git.sr.ht/~hedy/spsrv` and there will be a binary at `~/go/bin/` with the source code at `~/go/src/`
feel free to move the binary somewhere else like `/usr/sbin/`

### build it yourself
run `git clone https://git.sr.ht/~hedy/spsrv` from any directory and `cd spsrv`
make sure you have go installed and working.
```
go build
```
when it finishes, the binary will be in the current directory.

If you don't have/want go installed, you can contact me, and if you're lucky, I have the same OS as you and you can use my compiled binary (lol). I'll eventually have automated uploads of binaries for various architectures for each release in the future.

## configuration
The default config file location is `/etc/spsrv.conf` you can specify your own path by running spsrv like
```
spsrv -c /path/to/file.conf
```
You don't need a config file to have spsrv running, it will just use the default values.

### config options

Note that the options are case insensitive.

Here are the config options and their default values

**general**

- `port=300`: port to listen to
- `hostname="localhost"`: if this is set, any request that for hostnames other than this value would be rejected
- `rootdir="/var/spartan"`: folder for fetching files

**directory listing**

- `dirlistEnable=true`: enable directory listing for folders that does not have `index.gmi`
- `dirlistReverse=false`: reverse the order of which files are listed
- `dirlistSort="name"`: how files are sorted, only "name", "size", and "time" are accepted. Defaults to "name" if an unknown option is encountered
- `dirlistTitles=true`: if true, directory listing will use first top level header in `*.gmi` files instead of the filename

**~user/ directories**

- `userdirEnable=true`: enable serving `/~user/*` requests
- `userdir="public_spartan"`: root directory for users. This should not have trailing slashes, and it is relative to `/home/user/`

**CGI**

- `CGIPaths=["cgi/"]`: list of paths where world-executable files will be run as CGI processes. These paths would be checked if it prefix the requested path. For the default value, a request of `/cgi/hi.sh` (requesting to `./public/cgi/hi.sh`, for example) will run `hi.sh` script if it's world executable.
- `usercgiEnable=false`: enable running user's CGI scripts too. This is dangerous as spsrv does not (yet) change the Uid of the CGI process, hence the process would be ran by the same user that is running the server, which could mean write access to configuration files, etc. Note that this option will be assumed `false` if `userdirEnable` is set to `false`. Which means if user directories are not enabled, there will be no per-user CGI.

## todo
- [x] /folder to /folder/ redirects
- [x] directory listing
- [ ] logging to files
- [x] ~user directories
- [x] refactor working dir part
- [x] config
  - [ ] status meta
  - [x] user homedir
  - [x] hostname, port
  - [x] public dir
  - [ ] dirlist title
  - [ ] userdir slug
- [x] CGI
  - [x] pipe data block
  - [ ] user cgi config and change uid to user
  - [ ] regex in cgi paths
