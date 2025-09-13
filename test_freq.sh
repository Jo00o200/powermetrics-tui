#!/bin/bash
echo "Testing frequency sampler..."
sudo powermetrics --samplers cpu_power -n 1 -i 1000 2>/dev/null | grep -E "(MHz|frequency|Freq|cluster)" -i