#!/bin/bash

# register all devices detected as bcache devs
for dev in $(bin/lsblk -no path); do bcachectl super $dev && bcachectl register $dev; done
