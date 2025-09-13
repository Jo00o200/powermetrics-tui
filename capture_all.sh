#!/bin/bash
# Capture powermetrics output with all samplers
sudo powermetrics --samplers all -i 1000 -n 1 > /tmp/powermetrics_all_test.txt 2>&1
echo "Output saved to /tmp/powermetrics_all_test.txt"
grep -A 10 "tasks" /tmp/powermetrics_all_test.txt || echo "No tasks section found"