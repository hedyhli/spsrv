# spsrv

A static spartan server with many features:

* folder redirects
* /~user directories
* directory listing
* CONF or TOML config file
  * directory listing options
  * user directory feature and userdir path
* CGI

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
  - [ ] pipe data block
  - [ ] user cgi config and change uid to user
  - [ ] regex in cgi paths
