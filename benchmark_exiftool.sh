#!/bin/bash

TEST_DIR="/Users/antoinebedos/ownapps/ANTIGRAVITY/LOOT/CARD_VIRTUEL/A005"
EXIFTOOL_SCRIPT="/var/folders/l4/d5krzch95nv2220hj6wtsx_40000gn/T/loot_exiftool/exiftool"

echo "=== Benchmark: Direct ExifTool Performance ==="
echo "Testing on all R3D files in: $TEST_DIR"

if [ ! -f "$EXIFTOOL_SCRIPT" ]; then
  echo "Error: ExifTool script not found at $EXIFTOOL_SCRIPT"
  # Try to find it again via loot run if possible or just skip
  exit 1
fi

files=$(find "$TEST_DIR" -name "*.R3D")
count=$(echo "$files" | wc -l)
echo "Found $count files."

start_time=$(date +%s%N)
for f in $files; do
  perl "$EXIFTOOL_SCRIPT" -j -n -API LargeFileSupport=1 "$f" > /dev/null 2>&1
done
end_time=$(date +%s%N)

duration=$(( (end_time - start_time) / 1000000 ))
echo "Total time for $count files: ${duration}ms"
avg=$(( duration / count ))
echo "Average time per file: ${avg}ms"
