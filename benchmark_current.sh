#!/bin/bash

# Test avec 10 fichiers R3D
TEST_DIR="/Users/antoinebedos/ownapps/ANTIGRAVITY/LOOT/CARD_VIRTUEL/A005"

echo "=== Current Performance Baseline ==="
echo "Testing with files in: $TEST_DIR"

if [ ! -d "$TEST_DIR" ]; then
  echo "Error: Test directory $TEST_DIR does not exist."
  exit 1
fi

# Time metadata extraction only (using the embedded exiftool via go run to simulate internal call is hard, 
# so we will just run the exiftool binary if we can find it, or just time the main app dry run which does metadata)

echo "--- Benchmarking Metadata Extraction (via LOOT Dry Run) ---"
# LOOT's dry run now extracts metadata. We'll verify 10 files.
# We will use a subdirectory or just file arguments to limit to ~10 files if possible, 
# but for now let's just run on the A005 folder which likely has snippets.
time ./loot --dry-run "$TEST_DIR" /tmp/benchmark_test_dry > /dev/null

echo ""
echo "=== Full offload test (Copy + Metadata) ==="
# Time full offload
# We'll use a specific destination to avoid messing up previous tests
rm -rf /tmp/benchmark_test
time ./loot \
  --algorithm xxhash64 \
  "$TEST_DIR" \
  /tmp/benchmark_test > /dev/null
