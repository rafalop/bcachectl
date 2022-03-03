#!/bin/bash                                                                    
## Deploy one or more Ceph OSDs with bcache

DATA_DEVICES_STRING=""
DB_DEVICE=""
WAL_DEVICE=""
CACHE_DEVICE=""
CACHE_SIZE=""
DOIT=0
REUSE=0
CACHE_MODE=writethrough
SEQ_CUTOFF='8k'
BCACHECTL=/usr/local/bin/bcachectl
REQUIRED_PKGS=(
"parted"
"bcache-tools"
)
                                       
function print_help(){                 
  echo
  echo "Prepares one or more OSD(s) with cache on --cache-device and db on --db-device
                                                                                                                                                              
Required parameters:
  --data-devices {STRING}[,{STRING},{STRING}...]  the data device(s). Could be whole disk(s) or partition(s).
  --cache-device {STRING}  the cache device. Must be physical disk, we will try to add partition of size --cache-size to this device.
  --cache-size {STRING}  cache size per drive/osd
                                                                                                                                                              
Optional parameters:                    
  --data-devices {STRING},{STRING},{STRING} comma delimited list of devices to deploy, shares --cache-device and --db-device
  --db-device {STRING}  the db device to use for OSD rocksdb. Must be physical disk, we will try to add partition of size --db-size to this device.
  --db-size {STRING}  db size per drive/osd, required if --db-device is specified
  --wal-device {STRING}  the wal device to use for OSD wal. Must be physical disk, we will try to add partition of size --wal-size to this device.
  --wal-size {STRING}  wal size per drive/osd, required if --wal-device is specified
  --cache-mode  set writeback caching before deploying OSD (default writethrough)
  --seq-cutoff {string}  the bcache sequential cutoff tunable to set before deploy (default $SEQ_CUTOFF)
  --reuse  reuse partitions found with correct label (eg. PARTLABEL=\"sdX_cache\" or PARTLABEL=\"sdX_db\")
  --doit  actually execute

Examples:
$0 --data-devices /dev/sdb --cache-device /dev/sdd --cache-size 30G
$0 --data-devices /dev/sdb --cache-device /dev/sdd --cache-size 30G --db-device /dev/sdd --db-size 30G
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
    "--wal-device")
        WAL_DEVICE=${args[$pos]}
    ;;
    "--wal-size")
        WAL_SIZE=${args[$pos]}
    ;;
    "--cache-mode")
        CACHE_MODE=${args[$pos]}
    ;;                           
    "--seq-cutoff")
        SEQ_CUTOFF=${args[$pos]}
    ;;
    "--reuse")
        REUSE=1
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
if [[ "$DATA_DEVICES_STRING" == "" ]]
then 
  log error "Missing required parameter --data-devices"
  print_help
  exit 1
fi
if [[ "$CACHE_DEVICE" == "" ]]; then log error "Missing required parameter --cache-device";print_help;exit 1;fi
if [[ "$CACHE_SIZE" == "" ]]; then log error "Missing required parameter --cache-size";print_help;exit 1;fi

# Check dependent arguments
if [[ "$DB_DEVICE" != "" ]] && [[ "$DB_SIZE" == "" ]]; then log error "A DB device was specified but no --db-size was given"; print_help;exit 1;fi
if [[ "$DB_SIZE" != "" ]] && [[ "$DB_DEVICE" == "" ]]; then log error "A DB size was specified but no --db-device was given"; print_help;exit 1;fi
if [[ "$WAL_DEVICE" != "" ]] && [[ "$WAL_SIZE" == "" ]]; then log error "A WAL device was specified but no --wal-size was given"; print_help;exit 1;fi
if [[ "$WAL_SIZE" != "" ]] && [[ "$WAL_DEVICE" == "" ]]; then log error "A WAL size was specified but no --wal-device was given"; print_help;exit 1;fi

# Check overlapping devices
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

# Batch - multiple devices
if [[ ${#DATA_DEVICES[@]} -gt 1 ]]
then
  for dev in ${DATA_DEVICES[*]}
  do
    cmd="./$0 --data-devices $dev --cache-device $CACHE_DEVICE --cache-size $CACHE_SIZE --cache-mode $CACHE_MODE --seq-cutoff $SEQ_CUTOFF $(if [[ $DOIT -eq 1 ]];then echo "--doit";fi)"
    if [[ "$DB_DEVICE" != "" ]]
    then
      cmd="$CMD --db-device $DB_DEVICE --db-size $DB_SIZE" 
    fi
    if [[ "$WAL_DEVICE" != "" ]]
    then
      cmd="$CMD --wal-device $WAL_DEVICE --wal-size $WAL_SIZE" 
    fi
  done
  exit
else
  DATA_DEVICE=${DATA_DEVICES[0]}
fi

# Main
echo
log "==== bcache OSD Settings ===="
log "`printf "%-20s%s\n" "DATA_DEVICE:" "$DATA_DEVICE"`"
log "`printf "%-20s%s\n" "CACHE_DEVICE:" "$CACHE_DEVICE"`"
log "`printf "%-20s%s\n" "CACHE_SIZE:" "$CACHE_SIZE"`"
log "`printf "%-20s%s\n" "DB_DEVICE:" "$DB_DEVICE"`"
log "`printf "%-20s%s\n" "DB_SIZE:" "$DB_SIZE"`"
log "`printf "%-20s%s\n" "WAL_DEVICE:" "$WAL_DEVICE"`"
log "`printf "%-20s%s\n" "WAL_SIZE:" "$WAL_SIZE"`"
log "`printf "%-20s%s\n" "CACHE_MODE:" "$CACHE_MODE"`"
log "`printf "%-20s%s\n" "SEQ_CUTOFF:" "$SEQ_CUTOFF"`"


SHORTNAME=""
function get_shortname(){
  SHORTNAME=`echo $1 | sed -e 's/.*\/\(.*\)/\1/g'`  
}

log "Checking supplied parameters..."
# Check devices are ok to use (real devices, no filesystems)
if [[ ! -b $DATA_DEVICE ]]
then
  log error "$DATA_DEVICE is not a detected device (looked in /dev)."
  exit 1
fi

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

get_shortname $WAL_DEVICE
if [[ "$WAL_DEVICE" != "" ]] && [[ ! -d /sys/block/${SHORTNAME} ]]
then
  log error "${WAL_DEVICE} is not an acceptable DB device (physical disk that can be partitioned). Is it a partition?"
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
if [[ `blkid -o device -t PARTLABEL="${SHORTNAME}_cache"` ]]
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

# Check if wal partition already exists
get_shortname $WAL_DEVICE
if [[ "$WAL_DEVICE" != "" ]] && [[ `blkid -o device -t PARTLABEL="${SHORTNAME}_wal"` ]]
then
  log error "There already appears to be a wal partition for $SHORTNAME:"
  log error "$(blkid -t PARTLABEL=\"${SHORTNAME}_wal\")" 
  if [[ $REUSE -eq 0 ]];then exit 1;fi
fi

log "Checks complete"

if [[ $DOIT -ne 1 ]]
then
  log ""
  log "--doit was not used, exiting" 
  exit 0
fi

# Format the backing device
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
cmd="partx -a $CACHE_DEVICE"
runcmd "$cmd"
cache_dev=$(blkid -o device -t PARTLABEL="${SHORTNAME}_cache")
echo "cache_dev: $cache_dev"
if [[ "$cache_dev" != "" ]]
then
  log "Existing partition found for ${SHORTNAME}_cache."
  if [[ $REUSE -eq 0 ]]
  then
    log "--reuse was not specified, exiting as there is an existing partition cache partition (${cache_dev}) we are not going to reuse."
    exit 1
  fi
  $BCACHECTL stop $cache_dev 
else
  cmd="sgdisk -n 0:0:+${CACHE_SIZE} -c 0:${SHORTNAME}_cache $CACHE_DEVICE"
  runcmd "$cmd"
  cmd="partx -a $CACHE_DEVICE"
  runcmd "$cmd"
  if [[ ! `blkid -t PARTLABEL="${SHORTNAME}_cache"` ]]
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
runcmd "$cmd"
cmd="$BCACHECTL tune $DATA_DEVICE sequential_cutoff:$SEQ_CUTOFF"
runcmd "$cmd"

# Add optional db partition
get_shortname $DATA_DEVICE
if [[ "$DB_DEVICE" != "" ]]
then
  db_part=$(blkid -o device -t PARTLABEL="${SHORTNAME}_db")
  if [[ "$db_part" != "" ]]
  then
    log "Existing partition found for ${SHORTNAME}_db"
    if [[ $REUSE -eq 0 ]]
    then
      log "--reuse was not specified, exiting as there is an existing db partition (${db_part}) we are not going to reuse."
      exit 1
    fi
    ceph-volume lvm zap $db_part
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

# Add optional wal partition
get_shortname $DATA_DEVICE
if [[ "$WAL_DEVICE" != "" ]]
then
  wal_part=$(blkid -o device -t PARTLABEL="${SHORTNAME}_wal")
  if [[ "$wal_part" != "" ]]
  then
    log "Existing partition found for ${SHORTNAME}_wal"
    if [[ $REUSE -eq 0 ]]
    then
      log "--reuse was not specified, exiting as there is an existing wal partition (${wal_part}) we are not going to reuse."
      exit 1
    fi
    ceph-volume lvm zap $wal_part
  else
    log "Preparing wal partition..."
    get_shortname $DATA_DEVICE
    cmd="sgdisk -n 0:0:+${WAL_SIZE} -c 0:${SHORTNAME}_wal $WAL_DEVICE"
    runcmd "$cmd"
    cmd="partx -a $WAL_DEVICE"
    runcmd "$cmd"
  fi
  if ! blkid -t PARTLABEL="${SHORTNAME}_wal"
  then
    log error "Error setting up wal partition."
    if [[ $DOIT -eq 1 ]]; then exit 1; fi
  fi
fi


# Deploy OSD
log "Deploying OSD..."
get_shortname $DATA_DEVICE
osd_data_device=`$BCACHECTL list | grep $DATA_DEVICE | awk '{print $1}'`
osd_db_part=`blkid -o device -t PARTLABEL="${SHORTNAME}_db"`
osd_wal_part=`blkid -o device -t PARTLABEL="${SHORTNAME}_wal"`
cmd="ceph-volume lvm create --data $osd_data_device"
if [[ "$osd_db_part" != "" ]]; then cmd="$cmd --block.db $osd_db_part";fi
if [[ "$osd_wal_part" != "" ]]; then cmd="$cmd --block.wal $osd_wal_part";fi
runcmd "$cmd"
