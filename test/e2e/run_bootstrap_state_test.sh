#!/usr/bin/env bash
#
# This is a convenience script that takes a list of testnet manifests
# as arguments and runs each one of them sequentially. If a testnet
# fails, the container logs are dumped to stdout along with the testnet
# manifest, but the remaining testnets are still run.
#
# This is mostly used to run generated networks in nightly CI jobs.


./build/runner -f networks/bootstrap_state.toml&
PID=$!
echo $PID
#Sleeping to give time to validator05 to boot up 
sleep 20

docker logs -f validator05 |  sed '/Snapshot restored/ q'

h=$((`docker logs validator05  | grep " Snapshot restored" | awk -F '=' '{print $3}' | awk -F ' ' '{print $1}'`)); 

echo "Statesync done, stopping node at height "$h

#If validator05 is not booted up yet, this will error with "not found container validator05 but this is ok"
docker stop validator05

echo "Node stopped"


gsed -i "/from the height/{ n; s/enable = true/statesync_offline_height=$h/g }" networks/bootstrap_state/validator05/config/config.toml
rm -rf networks/bootstrap_state/validator05/data/blockstore.db
rm -rf networks/bootstrap_state/validator05/data/state.db
echo "Starting up validator..."
docker start validator05

docker logs -f validator05 |  sed '/bootstrapping done/ q'


echo "Bootstrap is finished "

echo "Validator started.. "

gsed -i "s/statesync_offline_height=$h/enable = false/g" networks/bootstrap_state/validator05/config/config.toml

docker start validator05

docker logs -f validator05 |  sed '/ersion info/ q'

docker logs -f validator05 | grep "Version info" -B 4

wait $PID

