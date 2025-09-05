#!/bin/bash

# download your liked songs, convert to mp3 with quality 3 [1 is highest, 10 is lowest], and embedd covers with a width of 200px

./main -url https://music.navidrome.example/rest -user jon -pass doe -mp3 -quality 3 -coversize 200

###########
# OPTIONAL STUFF, tells you the space used by your converted and/or covered songs, and the average bit_rate over 10 songs
###########

echo "space on device used $(du -hs rocksonic_songs/favs)"

COUNT=0
TOTAL=0
for song in rocksonic_songs/favs/*.mp3; do
  if [ $COUNT -gt 9 ]; then
    break
  fi
  ((COUNT = COUNT + 1))
  SIZE=$(ffprobe "$song" -show_streams 2>/dev/null | grep -iE "^bit_rate=[0-9]+$" | awk -F= '{ print($2) }')
  ((TOTAL = TOTAL + SIZE))
done
((TOTAL = TOTAL / COUNT / 1024))
echo Average of ${TOTAL}kbps

# feel free to call "copy_to_devices.sh" here to have one step less!
