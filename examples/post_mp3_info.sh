#!/bin/bash
# pass the output of rocksonice through "tail -n 1" and pass it to this script in quotes
#######
# OUTPUT=$(./main XXXXX | tail -n 1)
# ./post_mp3_info.sh "$OUTPUT"
#######
echo "space on device used $(du -hs "$1" | cut -f1)"

COUNT=0
TOTAL=0
while IFS= read -r song; do
  if [ $COUNT -gt 9 ]; then
    break
  fi
  ((COUNT = COUNT + 1))
  SIZE=$(ffprobe "$song" -show_streams 2>/dev/null | grep -iE "^bit_rate=[0-9]+$" | awk -F= '{ print($2) }')
  ((TOTAL = TOTAL + SIZE))
done < <(find "$1" -type f)
((TOTAL = TOTAL / COUNT / 1024))
echo Average of ${TOTAL}kbps
