#!/usr/bin/env bash
#
# This is a script that aims to simulate the scenario of out of band 
# statesync, performed by operators, offline state bootstrapping 
# and then finally booting up the node

# The script uses the networks/bootstrap_state.toml manifest as input
# Only validator05 is assumed to be running statesync. The node is stopped
# immediately after statesync is performed. (L24 ) The code has been altered 
# to statesync only one height and then return waiting for the node to be stopped

# Once the node is stopped we set a (new) config flag in config.toml:
# statesync_offline_height=h  - where h is the height at which the node had statesynced
# Statesync is disabled. The blockstore.db and state.db created with statesync are deleted
# The reason is that our statesync code will actually bootstrap the state store but in out-of-band
# statesync they are expected to be empty and bootstrapped separately.

# On startup the node will call BootstrapState (the code has been altered to perform this
# as the first statement of NewNode). The node will the return and shut down.

# The new config parameter has to be deleted from the config.toml of validator05 (L:54)

# The node is restarted. The node should start up normally, continuing to blocksync+consensus 
# and the test should terminate with an OK. 

make 
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

