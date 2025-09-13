#!/bin/bash

# This is the exact command the app runs to get powermetrics data
# Run this to capture a sample for debugging parser issues

echo "Capturing powermetrics output with all samplers..."
echo "This requires sudo access."
echo ""

# The exact command used by the app:
sudo powermetrics --samplers all -i 1000 -n 1 > sample_output.txt 2>&1

echo "Output saved to sample_output.txt"
echo ""
echo "First 50 lines of output:"
head -50 sample_output.txt

echo ""
echo "Checking for key sections..."
echo "- Interrupts: $(grep -c "IPI:" sample_output.txt) occurrences"
echo "- CPU Power: $(grep -c "CPU Power:" sample_output.txt) occurrences"
echo "- Processes: $(grep -c "^[0-9]" sample_output.txt | head -20) lines starting with numbers"
echo "- Thermal: $(grep -c "Thermal" sample_output.txt) occurrences"
echo "- Battery: $(grep -c "Battery" sample_output.txt) occurrences"