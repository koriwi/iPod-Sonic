#!/bin/bash

##########
# this script will sync your covered/converted songs to the first found known device.
# this example uses udisksctl,
# usually used by the major file managers to auto show/mount your removable disks.
# this script uses the same mechanism to mount your device without needing root permissions.
# then it uses rsync to synch your device with the output of rocksonic
##########

# put your devices block devices here, the script will be looking for them to mount them automatically
KNOWN_DEVICES=(
  /dev/disk/by-id/usb-SST55LD0_19K-45-C-MWE_100000000000A270018445AAA-0:0-part2
)
FOUND=""
for device in "${KNOWN_DEVICES[@]}"; do
  if ls "$device" &>/dev/null; then
    FOUND="$device"
    break
  fi
done

if [[ "$FOUND" == "" ]]; then
  echo "no known device found. exiting..."
  exit
fi
UDISKOUTPUT=$(udisksctl mount -b "${FOUND}" -o rw 2>&1)
if [ "$?" = "1" ]; then
  if echo "$UDISKOUTPUT" | grep already &>/dev/null; then
    echo "device already mounted. continuing"
  else
    exit
  fi
fi

MOUNT=$(echo "$UDISKOUTPUT" | awk -F'at ' '{print $2}' | sed 's/\.$//' | sed "s/'$//" | sed 's/`//')

# only update the existing files first, so we can sync smaller files and make some space
# we are copying the favs folder here, you need to change that if you selected a playlist instead of favs
rsync -rh ./rocksonic_songs/favs/ "$MOUNT/rocksonic/" --delete --update --size-only --info=progress2 --existing
rsync -rh ./rocksonic_songs/favs/ "$MOUNT/rocksonic/" --delete --update --size-only --info=progress2 --ignore-existing
sync
DEVICE=$(findmnt -n -o SOURCE "$MOUNT")
udisksctl unmount -b "$DEVICE"
