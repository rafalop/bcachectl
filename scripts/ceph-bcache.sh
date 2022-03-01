#!/bin/bash                                                                    
## Deploy one or more OSDs with bcache

DATA_DEVICE=""                         
DATA_DEVICES_STRING=""
DB_DEVICE=""
CACHE_DEVICE=""               
CACHE_SIZE=""              
DOIT=0                                                                         
REUSE=0
CACHE_MODE=writethrough
SEQ_CUTOFF='8k'
BCACHECTL=/usr/local/bin/bcachectl                                                                                                                             
REQUIRED_PKGS=(
"apache2"
"parted"
"bcache-tools"
)
                                       
function print_help(){                 
  echo
  echo "Prepares one or more OSD(s) with cache on --cache-device and db on --db-device
                                                                                                                                                              
Required parameters:
  --data-device {STRING} the data device. Could be whole disk or partition.
  --cache-device {STRING} the cache device. Must be physical disk, we will try to add partition of size --cache-size to this device.
  --cache-size {STRING} cache size per drive
                                                                                                                                                              
Optional parameters:                    
  --data-devices {STRING},{STRING},{STRING} comma delimited list of devices to deploy, shares --cache-device and --db-device
  --db-device {STRING} the db device to use for OSD rocksdb. Must be physical disk, we will try to add partition of size --db-size to this device.
  --db-size {STRING} 
  --reuse reuse partitions found with correct label (eg. PARTLABEL=\"sdX_cache\" or PARTLABEL=\"sdX_db\")
  --cache-mode set writeback caching before deploying OSD (default writethrough)
  --seq-cutoff {string} the bcache sequential cutoff tunable to set before deploy (default $SEQ_CUTOFF)
  --doit (actually execute)                                                                                                                                   

Examples:
$0 --data-device /dev/sdb --cache-device /dev/sdd --cache-size 30G
$0 --data-device /dev/sdb --cache-device /dev/sdd --cache-size 30G --db-device /dev/sdd --db-size 30G
$0 --data-devices /dev/sdb,/dev/sdc,/dev/sdd --cache-device /dev/nvme0n1 --cache-size 100G --db-device /dev/nvme0n1 --db-size 30G
                                                                                                                                                              
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
    "--data-device")
        DATA_DEVICE=${args[$pos]}
    ;;
    "--data-devices")
        DATA_DEVICES_STRING=${args[$pos]}
    ;;
    "--cache-device")
        CACHE_DEVICE=${args[$pos]}
    ;;
    "--cache-size")
        CACHE_SIZE=${args[$pos]}
    ;;
    "--db-device")
        DB_DEVICE=${args[$pos]}
    ;;
    "--db-size")                    
        DB_SIZE=${args[$pos]}
    ;;
    "--cache-mode")
        CACHE_MODE=${args[$pos]}
    ;;                           
    "--seq-cutoff")
        SEQ_CUTOFF=${args[$pos]}
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

# Check prerequisites
if [[ ! `which bcachectl` ]]
then
  echo "bcachectl is required for this script, and was not found. You can install it using:
curl https://raw.githubusercontent.com/rafalop/bcachectl/main/scripts/install_bcachectl.sh | bash
"
  exit 1
fi
for pkg in ${REQUIRED_PKGS[*]}
do
  if ! dpkg -s $pkg >/dev/null 2>&1
  then
    echo "The package '"$pkg"' is not installed. The following system packages are required for this script:"
    echo "${REQUIRED_PKGS[*]}"
    exit 1
  fi
done

function log() {
  if [[ "$1" == "debug" ]]
  then
    echo "--> DEBUG: $2"
  elif [[ "$1" == "error" ]]
  then
    echo "ERROR: $2"
  else
    echo "--> $1"
  fi
}

RET=0
function runcmd() {
  if [[ "$DOIT" -eq 1 ]]
  then
    echo "--> RUNNING: $1"
    $1
    RET=$?
  else
    echo "--> NOT RUNNING: $1"
  fi
}

# Check required arguments were given
if [[ "$DATA_DEVICE" != "" ]] && [[ "$DATA_DEVICES_STRING" != "" ]]
then
  log error "Can only specify one of --data-device (single) or --data-devices (multiple,devs)"
  print_help
  exit
fi
if [[ "$DATA_DEVICE" == "" ]] && [[ "$DATA_DEVICES_STRING" == "" ]]
then 
  log error "Missing required parameter --data-device or --data-devices"
  print_help
  exit 1
fi
if [[ "$CACHE_DEVICE" == "" ]]; then log error "Missing required parameter --cache-device";print_help;exit 1;fi
if [[ "$CACHE_SIZE" == "" ]]; then log error "Missing required parameter --cache-size";print_help;exit 1;fi

# Check dependent arguments
if [[ "$DB_DEVICE" != "" ]] && [[ "$DB_SIZE" == "" ]]; then log error "A DB device was specified but no --db-size was given"; print_help;exit 1;fi
if [[ "$DB_SIZE" != "" ]] && [[ "$DB_DEVICE" == "" ]]; then log error "A DB size was specified but no --db-device was given"; print_help;exit 1;fi

# Check overlapping devices
if [[ "$DATA_DEVICE" != "" ]] && [[ "$CACHE_DEVICE" == "$DATA_DEVICE" ]]; then log error "The data device and the cache device cannot be the same ($DATA_DEVICE was given for both)"; print_help; exit 1;fi
if [[ "$DATA_DEVICE" != "" ]] && [[ "$DB_DEVICE" == "$DATA_DEVICE" ]]; then log error "The data device and the db device cannot be the same ($DATA_DEVICE was given for both)"; print_help; exit 1;fi

if [[ "$DATA_DEVICES_STRING" != "" ]]
then
  DATA_DEVICES=( $(echo $DATA_DEVICES_STRING | tr ',' ' ') )
  for dev in ${DATA_DEVICES[*]}
  do
    if [[ "$dev" == $CACHE_DEVICE ]]
    then
      log error "The data device and cache device cannot be the same (${dev} was given for both)"
      exit 1
    fi
    if [[ "$dev" == $DB_DEVICE ]]
    then
      log error "The data device and db device cannot be the same (${dev} was given for both)"
      exit 1
    fi
  done
fi



if [[ "$DATA_DEVICES_STRING" != "" ]]
then
  log "Runing in batch mode (multiple data devices)..."
  log "==== Batch settings ===="
  log "`printf "%-20s%s\n" "DATA_DEVICES:" "$DATA_DEVICES_STRING"`"
  log "`printf "%-20s%s\n" "CACHE_DEVICE:" "$CACHE_DEVICE"`"
  log "`printf "%-20s%s\n" "CACHE_SIZE:" "$CACHE_SIZE"`"
  log "`printf "%-20s%s\n" "DB_DEVICE:" "$DB_DEVICE"`"
  log "`printf "%-20s%s\n" "DB_SIZE:" "$DB_SIZE"`"
  log "`printf "%-20s%s\n" "CACHE_MODE:" "$CACHE_MODE"`"
  log "`printf "%-20s%s\n" "SEQ_CUTOFF:" "$SEQ_CUTOFF"`"
  log ""
  DATA_DEVICES=( $(echo $DATA_DEVICES_STRING | tr ',' ' ') )
  for dev in ${DATA_DEVICES[*]}
  do
    log "Deploying $dev with cache on $CACHE_DEVICE and db on $DB_DEVICE..."
    if [[ "$DB_DEVICE" == "" ]]
    then
      bash ./$0 --data-device $dev --cache-device $CACHE_DEVICE --cache-size $CACHE_SIZE --cache-mode $CACHE_MODE --seq-cutoff $SEQ_CUTOFF $(if [[ $DOIT -eq 1 ]];then echo "--doit";fi)
    else
      bash ./$0 --data-device $dev --cache-device $CACHE_DEVICE --cache-size $CACHE_SIZE --db-device $DB_DEVICE --db-size $DB_SIZE --cache-mode $CACHE_MODE --seq-cutoff $SEQ_CUTOFF $(if [[ $DOIT -eq 1 ]];then echo "--doit";fi)
    fi
  done
  exit
fi

log "==== Settings ===="
log "`printf "%-20s%s\n" "DATA_DEVICE:" "$DATA_DEVICE"`"
log "`printf "%-20s%s\n" "CACHE_DEVICE:" "$CACHE_DEVICE"`"
log "`printf "%-20s%s\n" "CACHE_SIZE:" "$CACHE_SIZE"`"
log "`printf "%-20s%s\n" "DB_DEVICE:" "$DB_DEVICE"`"
log "`printf "%-20s%s\n" "DB_SIZE:" "$DB_SIZE"`"
log "`printf "%-20s%s\n" "CACHE_MODE:" "$CACHE_MODE"`"
log "`printf "%-20s%s\n" "SEQ_CUTOFF:" "$SEQ_CUTOFF"`"

if [[ $DOIT -ne 1 ]]
then
  log ""
  log "--doit was not used, exiting" 
  exit 0
fi

SHORTNAME=""
function get_shortname(){
  SHORTNAME=`echo $1 | sed -e 's/.*\/\(.*\)/\1/g'`  
}

log "Checking supplied parameters..."
# Check devices are ok to use (real devices, no filesystems)
get_shortname $CACHE_DEVICE
if [[ ! -d /sys/block/${SHORTNAME} ]]
then
  log error "${CACHE_DEVICE} is not an acceptable CACHE device (physical disk that can be partitioned). Is it a partition?"
  exit 1
fi

get_shortname $DB_DEVICE
if [[ "$DB_DEVICE" != "" ]] && [[ ! -d /sys/block/${SHORTNAME} ]]
then
  log error "${DB_DEVICE} is not an acceptable DB device (physical disk that can be partitioned). Is it a partition?"
  exit 1
fi

# Check data device is not already a bcache device
if $BCACHECTL list | grep $DATA_DEVICE
then
  log error "$DATA_DEVICE is already a bcache device, exiting."
  exit 1
fi

# Check if a cache already exists for the backing device
get_shortname $DATA_DEVICE
if blkid -o device -t PARTLABEL="${SHORTNAME}_cache"
then
  log error "There already appears to be a cache partition for $SHORTNAME:"
  log error "$(blkid -t PARTLABEL=\"${SHORTNAME}_cache\")"
  if [[ $REUSE -eq 0 ]];then exit 1;fi
fi

# Check if db partition already exists
get_shortname $DB_DEVICE
if [[ "$DB_DEVICE" != "" ]] && [[ `blkid -o device -t PARTLABEL="${SHORTNAME}_db"` ]]
then
  log error "There already appears to be a db partition for $SHORTNAME:"
  log error "$(blkid -t PARTLABEL=\"${SHORTNAME}_db\")" 
  if [[ $REUSE -eq 0 ]];then exit 1;fi
fi

# Format the backing device
log "Checks complete"
log "Preparing bcache device..."
cmd="$BCACHECTL add -B $DATA_DEVICE"
runcmd "$cmd"
if [[ $RET -ne 0 ]]
then
  log error "Error creating the bcache device."
  if [[ $DOIT -eq 1 ]]; then exit 1 ;fi
fi

# Add cache partition
log "Preparing cache partition..."
get_shortname $DATA_DEVICE
cache_dev=$(blkid -o -t PARTLABEL="${SHORTNAME}_cache")
if [[ "$cache_dev" != "" ]]
then
  log "Existing partition found for ${SHORTNAME}_cache, attempting to reuse."
  $BCACHECTL stop $cache_dev 
else
  cmd="sgdisk -n 0:0:+${CACHE_SIZE} -c 0:${SHORTNAME}_cache $CACHE_DEVICE"
  runcmd "$cmd"
  cmd="partx -a $CACHE_DEVICE"
  runcmd "$cmd"
  if ! blkid -t PARTLABEL="${SHORTNAME}_cache"
  then
    log error "Error creating cache partition."
    if [[ $DOIT -eq 1 ]]; then exit 1; fi
  fi
fi
cmd="$BCACHECTL add -C $(blkid -o device -t PARTLABEL=\"${SHORTNAME}_cache\") --wipe-super"
runcmd "$cmd"

# Attach cache to backing device
get_shortname $DATA_DEVICE
cache_dev=$(blkid -o device -t PARTLABEL="${SHORTNAME}_cache")
log "Attaching cache device $cache_dev to backing device $DATA_DEVICE"
cmd="$BCACHECTL attach $cache_dev $DATA_DEVICE"
runcmd "$cmd"
cmd="$BCACHECTL tune $DATA_DEVICE cache_mode:$CACHE_MODE"
runcmd
cmd="$BCACHECTL tune $DATA_DEVICE sequential_cutoff:$SEQ_CUTOFF"
runcmd

# Add optional db partition
get_shortname $DATA_DEVICE
if [[ "$DB_DEVICE" != "" ]]
then
  db_dev=$(blkid -o -t PARTLABEL="${SHORTNAME}_cache")
  if [[ "$db_dev" != "" ]]
  then
    log "Existing partition found for ${SHORTNAME}_cache, attempting to reuse."
    ceph-volume lvm zap $db_dev
  else
    log "Preparing db partition..."
    get_shortname $DATA_DEVICE
    cmd="sgdisk -n 0:0:+${DB_SIZE} -c 0:${SHORTNAME}_db $DB_DEVICE"
    runcmd "$cmd"
    cmd="partx -a $DB_DEVICE"
    runcmd "$cmd"
  fi
  if ! blkid -t PARTLABEL="${SHORTNAME}_db"
  then
    log error "Error setting up db partition."
    if [[ $DOIT -eq 1 ]]; then exit 1; fi
  fi
fi


# Deploy OSD
log "Deploying OSD..."
get_shortname $DATA_DEVICE
osd_data_device=`$BCACHECTL list | grep $DATA_DEVICE | awk '{print $1}'`
osd_db_device=`blkid -o device -t PARTLABEL="${SHORTNAME}_db"`
cmd="ceph-volume lvm create --data $osd_data_device --block.db $osd_db_device"
runcmd "$cmd"