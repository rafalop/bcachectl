# bcachectl
This tool makes use of the sysfs and paths that are installed by bcache tools when you format/register devices to make it much simpler to view, configure and modify bcache setups.

## Requirements
You should have a kernel that supports bcache (eg. Ubuntu 18.04+), and `bcache-tools` package installed.

## Building binary
First install golang: https://golang.org/doc/install
Then
```
go mod init bcachectl
go mod tidy
go build bcachectl.go
```
This will produce the binary `bcachectl` in the same dir. You can place in /usr/local/bin or other path of your choice.

## Usage examples
### Format and register a bcache backing device
`bcachectl format -B /dev/vdb`
### Format and register a bcache cache device
`bcachectl format -C /dev/vdc`
### Format and register a bcache device together with a cache device (auto attaches the cache)
`bcachectl format -B /dev/vdb -C /dev/vdc`
### List all bcache devices
```
bcachectl list
bcachectl -f json list
bcachectl -f short list
```
### Show detailed information about a bcache device
```
bcachectl show /dev/vdb
bcachectl show bcache0
```
### Unregister a bcache device
bcachectl unregister bcache0
