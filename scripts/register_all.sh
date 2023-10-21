#!/bin/bash

# register all devices detected as bcache devs
echo "looking for bcache devices to register..."
for dev in $(bcachectl.lsblk -no path); do bcachectl super $dev >/dev/null 2>&1 && bcachectl register $dev; done
exit 0
