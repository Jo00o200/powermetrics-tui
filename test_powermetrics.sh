#!/bin/bash

echo "=== Testing different samplers ==="

echo -e "\n--- CPU Power ---"
sudo powermetrics --samplers cpu_power -n 1 -i 1000 2>/dev/null | head -30

echo -e "\n--- Thermal ---"
sudo powermetrics --samplers thermal -n 1 -i 1000 2>/dev/null | head -30

echo -e "\n--- SMC (for temperature) ---"
sudo powermetrics --samplers smc -n 1 -i 1000 2>/dev/null | head -30

echo -e "\n--- Battery ---"
sudo powermetrics --samplers battery -n 1 -i 1000 2>/dev/null | head -30

echo -e "\n--- All samplers ---"
sudo powermetrics --samplers all -n 1 -i 1000 2>/dev/null | grep -E "(Power|power|Temperature|temperature|Battery|battery|Thermal|thermal)" | head -20