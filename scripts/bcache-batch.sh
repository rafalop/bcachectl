#!/bin/bash
## Script for batch setting up of bcache partitions


DATA_DRIVES=""
CACHE_DEVICE=""
CACHE_SIZE=""
WIPE_DRIVES=0
UNREGISTER_ALL=0
DOIT=0
BCACHECTL=/usr/local/bin/bcachectl
REQUIRED_PKGS=(
"parted"
"bcache-tools"
)

function print_help(){
  echo "Prepares batch of data disk(s) with caches on a single cache device using bcache.

Required arguments:
  --data-drives '{STRING}' regexp string to capture data drives, examples: \"9.1T\", \"sd[b-g]\")
  --cache-device {STRING} flash drive to carve up to use for caches for specified data drives, eg. /dev/nvme0n1)
  --cache-size {STRING} cache size per drive)

Optional arguments:
  --wipe-drives (wipe all drives before starting, this will destroy all data on the drives!!)
  --unregister-all (will unregister ALL current bcache devices, regardless of other flags - USE WITH CAUTION!)
  --doit (actually execute)

"
}

args=($@)
for arg in ${args[*]}
do
    pos=$(($count+1))
    case $arg in
    "--doit")
        DOIT=1
    ;;
    "--data-drives")
        DATA_DRIVES=(${args[$pos]})
    ;;
    "--cache-device")
        CACHE_DEVICE=${args[$pos]}
    ;;
    "--cache-size")
        CACHE_SIZE=${args[$pos]}
    ;;
    "--wipe-drives")
        WIPE_DRIVES=1
    ;;
    "--unregister-all")
        UNREGISTER_ALL=1
    ;;
    "-h")
        print_help
        exit
    ;;
    "-help")
        print_help
        exit
    ;;
        *)
        :
    ;;
    esac
    count=$(($count+1))
done


## Check required packages
pkgs_ok=0
for pkg in ${REQUIRED_PKGS[*]}
do
  if [[ ! `dpkg -s $pkg 2>/dev/null` ]]
  then
    echo "$pkg is not installed."
    pkgs_ok=1
  fi
  if [[ $pkgs_ok -eq 1 ]]
  then
    echo -e "One or more packages required by this script are missing.\nRequired packages are: ${REQUIRED_PKGS[*]}"
    exit 1
  fi 
done

## Checking of arguments
if [[ "$DATA_DRIVES" == "" ]] || [[ "$CACHE_DEVICE" == "" ]] || [[ "$CACHE_SIZE" == "" ]]
then
  echo
  echo "You must provide at least 3 arguments: --data-drives, --cache-device, --cache-size"
  print_help
  exit 1
fi

captured_data_drives=`lsblk -l | egrep "$DATA_DRIVES" | awk '{print $1}' | sed -e 's/^/\/dev\//g' | xargs`
cdd=($captured_data_drives)

printf "Data drives capture regexp: %s (captures %s)\n" "${DATA_DRIVES}" "${cdd[*]}"
printf "Cache device: %s" ${CACHE_DEVICE}
if [[ "$CACHE_DEVICE" != "/dev/"* ]];then printf " (Trying /dev/${CACHE_DEVICE})\n";CACHE_DEVICE=/dev/${CACHE_DEVICE};else printf "\n";fi 
printf "Cache size (per drive): %-30s\n" $CACHE_SIZE

## Insert pause to proceed

function run_command(){
  if [[ $DOIT -eq 1 ]]
  then
    echo "RUNNING: $1"
    $1 
  else
    echo "NOT RUNNING: $1"
  fi
}

# Main
if [[ $DOIT -ne 1 ]]
then 
  echo "--doit was not used, only printing commands" 
  echo
fi

## Basic checks
# Check if cache device is partition otherwise it can't support multiple data drives
if [[ `echo "$CACHE_DEVICE" | egrep '[0-9]+'` ]] && [[ ${#cdd[@]} -gt 1 ]]
then
  echo "Cache device appears to be a partition, and cannot be used for multiple data drives. Exiting."
  exit 1
fi
# Check if any listed devices here are already in use by bcache
for drive in ${cdd[@]} ${CACHE_DEVICE}
do
  if $BCACHECTL list | grep -q $drive
  then
    if [[ $UNREGISTER_ALL -eq 1 ]]
    then
      $BCACHECTL list | egrep '^/dev' | awk '{print $1}' | xargs -I {} bash -c "$BCACHECTL unregister {}"
    else
      echo "One of your devices (${drive}) is still registered and in use by bcache, it must be unregistered before being able to be used."
      echo "try:
$BCACHECTL list
$BCACHECTL unregister $drive"
      exit 1
    fi
  fi
done


## Wipe drives
if [[ $WIPE_DRIVES -eq 1 ]]
then
  echo "Wiping drives..."
  for drive in ${cdd[@]} ${CACHE_DEVICE}
  do
    run_command "wipefs -a $drive"
    run_command "dd if=/dev/zero of=${drive} bs=1M count=1 conv=sync"
    ## If raw disk (not partition) also zap
    if [[ ! `echo $drive | egrep '[0-9]+'` ]]
    then
      run_command "sgdisk -Z $drive"
    fi
  done
  echo
fi

## If we have more than one data drive, assume we have to add partitions
## to the cache drive per data drive
echo "Preparing cache partitions..."
if [[ ! `echo $CACHE_DEVICE | egrep '[0-9]+'` ]]
then
  for cache_slot in ${cdd[@]}
  do
    short_name=`echo $cache_slot | cut -d'/' -f3`
    run_command "sgdisk -n 0:0:+${CACHE_SIZE} -c 0:${short_name}_cache ${CACHE_DEVICE}"
    run_command "partprobe $CACHE_DEVICE"
  done
else
  echo "cache device is a partition, nothing to prepare."
fi
echo

## Make sure devs show up in blkid
if [[ $DOIT -eq 1 ]]
then
  sleep 5
fi

## Set up bcache
for drive in ${cdd[@]}
do
  short_name=`echo $drive | cut -d'/' -f3`
  #echo "short_name: $short_name"
  cache_dev=`blkid -t PARTLABEL="${short_name}_cache" | cut -d':' -f1`
  #echo "$cache_dev"
  if [[ "$cache_dev" == "" ]]
  then
    cache_dev='(unknown yet)'
    exit 1
  fi
  # unmount drives
  if [[ $WIPE_DRIVES == 1 ]]
  then
    $BCACHECTL unregister ${cache_dev} >/dev/null
  fi
  run_command "$BCACHECTL add -B ${drive} -C ${cache_dev} --wipe-bcache"
done

if [[ $DOIT = 1 ]]
then
  echo "Results:"
  $BCACHECTL list
fi

