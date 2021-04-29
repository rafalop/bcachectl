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
This will produce the binary `bcachectl` in the same dir. You can place in /usr/local/bin or other exec path of your choice.

## Usage examples
### Format and register a bcache backing device
`bcachectl add -B /dev/vdb`
### Format and register a bcache cache device
`bcachectl add -C /dev/vdc`
### Format and register a bcache device together with a cache device (auto attaches the cache)
`bcachectl add -B /dev/vdb -C /dev/vdc`
### List all bcache devices
```
bcachectl list
bcachectl list -f json
bcachectl list -f short
```
### Show detailed information about a bcache device
```
bcachectl show /dev/vdb
bcachectl show bcache0
```

### Attach an already formatted cache dev to an already formatted backing dev
```
bcachectl attach /dev/ssd /dev/vda
```
### Detach a cache device (/dev/ssd) from a backing device (/dev/vda)
```
bcachectl detach /dev/sdd /dev/vda
```
### Change bcache tunable of a bcache device
```
bcachectl tune bcache0 cache_mode:writeback
bcachectl tune /dev/vdb sequential_cutoff:$((1024*1024))
```

