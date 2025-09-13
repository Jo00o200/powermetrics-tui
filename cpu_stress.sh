#!/bin/bash
# Simple CPU stress test for Mac

echo "Starting CPU stress test..."
echo "Press Ctrl+C to stop"

# Number of CPU cores to stress (default: all cores)
CORES=${1:-$(sysctl -n hw.ncpu)}

echo "Stressing $CORES CPU cores..."

# Launch background processes that consume CPU
for i in $(seq 1 $CORES); do
    yes > /dev/null &
done

# Wait for user to stop
wait
