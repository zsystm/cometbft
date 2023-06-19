#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

./build/load \
    -c 1 -T 60 -r 300 -s 1024 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/websocket

