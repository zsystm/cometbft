#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

./build/load \
    -c 1 -T 600 -r 1500 -s 4096 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/v1/websocket

