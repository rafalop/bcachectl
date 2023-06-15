#!/bin/bash

GO_VERSION='1.20.1'
AARCH64_GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz"
X86_GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
INSTALL_LOC=/usr/local/bin/bcachectl
TMPDIR=/var/tmp
REINSTALL=0

if [[ "$1" == "reinstall" ]]; then REINSTALL=1;fi

if [[ -f $INSTALL_LOC ]] && [[ REINSTALL -eq 0 ]]
then
        echo "bcachectl is already installed (${INSTALL_LOC})  If you want to reinstall, rerun this script:

$0 reinstall"
        exit 0
fi

if [[ ! `which git` ]]
then
        echo "this script requires 'git', please install before proceeding."
        exit 1
fi

# Check architecture
if [[ "`uname -m`" == "aarch64" ]]
then
        GO_URL=$AARCH64_GO_URL
elif [[ "`uname -m`" == "x86_64" ]]
then
        GO_URL=$X86_GO_URL
else
        echo "Unknown architecture `uname -m`"!
fi
GO_ZIP=`echo $GO_URL | sed -e 's/.*\/\(.*\)/\1/g'`

# Download and install go to temp location
cd $TMPDIR
if [[ ! -d .bcachectl_install ]]; then mkdir .bcachectl_install; fi
cd .bcachectl_install
if [[ ! -f $GO_ZIP ]]; then echo "No go zip found, downloading...";echo;wget $GO_URL ;fi
if [[ ! -d go ]]; then echo "Installing go to $PWD/go...";tar -xzf $GO_ZIP; else echo "Go found installed, not installing. ";fi 
GOBIN=${TMPDIR}/.bcachectl_install/go/bin/go

# Clone bcachectl repo
echo
echo "Getting latest version of bcachectl..."
if [[ ! -d bcachectl ]]; then git clone https://github.com/rafalop/bcachectl.git; else cd bcachectl && git pull && cd ..;fi

# Build bcachectl 
echo
echo "Building bcachectl..."
cd ${TMPDIR}/.bcachectl_install/bcachectl
if [[ ! -f go.mod ]]; then $GOBIN mod init bcachectl; fi
$GOBIN mod tidy
$GOBIN build bcachectl

# Install bcachectl
echo
if [[ -f bcachectl ]]
then
        echo "Installing bcachectl to $INSTALL_LOC..."
        cp bcachectl $INSTALL_LOC
        echo "Done!"
else
        echo "Error building bcachectl."
fi
