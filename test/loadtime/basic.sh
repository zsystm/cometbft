#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

for i in {1..60}; do

./build/load \
    -c 1 -T 60 -r 300 -s 2048 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/websocket

sleep 300

done
