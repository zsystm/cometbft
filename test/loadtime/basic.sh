#!/bin/sh

set -euo pipefail

# A basic invocation of the loadtime tool.

./build/load \
    -c 1 -T 9000 -r 1000 -s 8096 \
    --broadcast-tx-method sync \
    --endpoints ws://localhost:26657/v1/websocket

#sleep 10

#./build/load \
#    -c 1 -T 3600 -r 1000 -s 8096 \
#    --broadcast-tx-method sync \
#    --endpoints ws://localhost:26657/v1/websocket

#sleep 10

#./build/load \
#    -c 1 -T 3600 -r 1000 -s 8096 \
#    --broadcast-tx-method sync \
#    --endpoints ws://localhost:26657/v1/websocket
#
#sleep 30 


#kill -9 85110

#kill -9 85214 



