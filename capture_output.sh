#!/bin/bash

echo "Capturing powermetrics output..."
sudo powermetrics --samplers interrupts -i 1000 -n 1 > /tmp/powermetrics_output.txt 2>&1
echo "Output saved to /tmp/powermetrics_output.txt"
cat /tmp/powermetrics_output.txt