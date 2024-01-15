#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

./build/load \
    -c 1 -T 10 -r 10 -s 1048576 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/v1/websocket,ws://localhost:26660/v1/websocket 
