# spsrv

A static spartan server with many features:

* folder redirects
* /~user directories
* directory listing
* CONF or TOML config file
* CGI

Known servers running spsrv:
* [hedy.tilde.cafe:3333](https://portal.mozz.us/spartan/hedy.tilde.cafe:3333)
* [tilde.team](https://portal.mozz.us/spartan/tilde.team)
* [tilde.cafe](https://portal.mozz.us/spartan/tilde.cafe)
* [earthlight.xyz:3000](https://portal.mozz.us/spartan/earthlight.xyz:3000)
* [jdcard.com:3300](https://portal.mozz.us/spartan/jdcard.com:3300/)
* [do.hedy.dev](https://portal.mozz.us/spartan/do.hedy.dev/)

**Questions / Support**

* [#spartan on Tilde.Chat IRC](https://tilde.chat/kiwi/#spartan) (please ping
  hedy)
* [Public inbox](mailto:~hedy/inbox@lists.sr.ht) (general mailing list on
  lists.sr.ht)
* Patches: { [~hedy/inbox at lists.sr.ht](https://lists.sr.ht/~hedy/inbox) }

---

**Table of contents**

<!-- vim-markdown-toc GFM -->

* [install](#install)
    * [Option 1: prebuilt binaries](#option-1-prebuilt-binaries)
    * [Option 2: with `go install`](#option-2-with-go-install)
    * [Option 3: just build it yourself](#option-3-just-build-it-yourself)
    * [otherwise...](#otherwise)
* [configuration](#configuration)
    * [config options](#config-options)
* [CLI](#cli)
* [CGI](#cgi)
* [Help / Issues / Feedback](#help--issues--feedback)
* [todo](#todo)

<!-- vim-markdown-toc -->

## install

you have three options:

### Option 1: prebuilt binaries

prebuilt binaries for darwin and linux architectures arm/amd-64 are provided
since v0.5.4. Head over to the [tags page on
git.sr.ht](https://git.sr.ht/~hedy/spsrv/refs), click on a desired tag and
download the binary for your architecture.


### Option 2: with `go install`

first, you need to have go installed and have a folder `~/go` with `$GOPATH`
pointing to it.

```
go install git.sr.ht/~hedy/spsrv@latest
```

there will be a binary at `~/go/bin/` with the source code at `~/go/src/`

feel free to move the binary somewhere else like `/usr/sbin/`

note that it's recommended to pin any latest version `@v0.0.0` rather than the
latest commit since it may not be stable.


### Option 3: just build it yourself

run `git clone https://git.sr.ht/~hedy/spsrv` from any directory and `cd spsrv`

make sure you have go installed and working.

```
git checkout v0.0.0  # recommended to pin a specific tag
make build
```

when it finishes, the binary will be in `./bin`.

if you don't have make, you can just `go build` (just that version and build
information will not be available with `spsrv --version`).

### otherwise...

if you do not wish to install go or clone the repo, and your architecture is not
supported in the prebuilt binaries, drop an email to my [public
inbox](mailto:~hedy/inbox@lists.sr.ht) (or contact me privately) so I could
perhaps compile a binary for your architecture.


## configuration

The default config file location is `/etc/spsrv.conf` you can specify your own
path by running spsrv like

```
spsrv -c /path/to/file.conf
```

You don't need a config file to have spsrv running, it will just use the
default values.


### config options

Note that the options are case insensitive.

Here are the config options and their default values

**general**

* `port=300`: port to listen to
* `hostname="localhost"`: if this is set, any request that for hostnames other than this value would be rejected
* `rootdir="/var/spartan"`: folder for fetching files

**directory listing**

* `dirlistEnable=true`: enable directory listing for folders that does not have `index.gmi`
* `dirlistReverse=false`: reverse the order of which files are listed
* `dirlistSort="name"`: how files are sorted, only "name", "size", and "time" are accepted. Defaults to "name" if an unknown option is encountered
* `dirlistTitles=true`: if true, directory listing will use first top level header in `*.gmi` files instead of the filename

**~user/ directories**

* `userdirEnable=true`: enable serving `/~user/*` requests
* `userdir="public_spartan"`: root directory for users. This should not have trailing slashes, and it is relative to `/home/user/`
* `userSubdomains=false`: User vhosts. Whether to allow `user.host.name/foo.txt` being the same as `host.name/~user/foo.txt` (When `hostname="host.name"`). **NOTE**: This only works when `hostname` option is set.

**CGI**

* `CGIPaths=["cgi/"]`: list of paths where world-executable files will be run as CGI processes. These paths would be checked if it prefix the requested path. For the default value, a request of `/cgi/hi.sh` (requesting to `./public/cgi/hi.sh`, for example) will run `hi.sh` script if it's world executable.
* `usercgiEnable=false`: enable running user's CGI scripts too. This is dangerous as spsrv does not (yet) change the Uid of the CGI process, hence the process would be ran by the same user that is running the server, which could mean write access to configuration files, etc. Note that this option will be assumed `false` if `userdirEnable` is set to `false`. Which means if user directories are not enabled, there will be no per-user CGI.

Check out some example configuraton in the [examples/](examples/) directory.

## CLI

You can override values in config file if you supply them from the command line:

```
Usage: spsrv [ [ -c <path> -h <hostname> -p <port> -d <path> ] | --help | --version ]

    -c, --config string     Path to config file
    -d, --dir string        Root content directory
    -h, --hostname string   Hostname
    -p, --port int          Port to listen to
```

Note that you *cannot* set the hostname or the dir path to `,` because spsrv
uses that to check whether you provided an option. You can't set port to `0`
either, sorry, this limitation comes with the advantage of being able to
override config values from the command line.

There are no arguments wanted when running spsrv, only options as listed above :)

## CGI

The following environment values are set for CGI scripts:

```
GATEWAY_INTERFACE # CGI/1.1
REMOTE_ADDR      # Remote address
SCRIPT_PATH      # (Relative) path of the CGI script
SERVER_SOFTWARE  # SPSRV
SERVER_PROTOCOL  # SPARTAN
REQUEST_METHOD   # Set to nothing
SERVER_PORT      # Port
SERVER_NAME      # Hostname
DATA_LENGTH      # Input data length
```

The data block, if any, will be piped as stdin to the CGI process.

Keep in mind that CGI scripts (as of now) are run by the same user as the
server process, hence it is generally dangerous for allowing users to have
their own CGI scripts. See configuration section for more details.

Check out some example CGI scripts in the [examples/](examples/) directory.

Example systemd service configurations are also listed there. Feel free to
contribute for other OSes :)


## Help / Issues / Feedback

Please either use the [#spartan channel on tilde.chat
IRC](https://tilde.chat/kiwi/#spartan) or my [public
inbox](https://lists.sr.ht/~hedy/inbox).

Both are listed at the top of this document.

**Patches** -> [public inbox](https://lists.sr.ht/~hedy/inbox)


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
  - [x] dirlist title
  - [x] user vhost
  - [ ] userdir slug
  - [ ] redirects
- [x] CGI
  - [x] pipe data block
  - [ ] user cgi config and change uid to user
  - [ ] regex in cgi paths
- [ ] SCGI

- [ ] Multiple servers with each of their own confs

README:
- [x] Add example confs (added in [examples/](examples) directory)
- [x] Add example .service files (added in [examples/](examples) directory)
