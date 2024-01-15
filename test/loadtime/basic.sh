#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

./build/load \
    -c 1 -T 9000 -r 1000 -s 8096 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/v1/websocket


sleep 100 

kill -9 83719

kill -9 83796
