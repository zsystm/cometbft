#!/usr/bin/env bash

set -ex

NUM_NODES=8

# Start Prometheus server (if it's not already running)
./build/runner -f networks/${NUM_NODES}nodes_push.toml setup
prometheus --config.file=networks/${NUM_NODES}nodes_push/prometheus.yaml > /dev/null &
sleep 2

MANIFESTS_DIR=networks/tmp
mkdir -p $MANIFESTS_DIR
LOGS_DIR=networks/logs/`date +"%FT%H%M%z"`
mkdir -p $LOGS_DIR

# Set parameters
declare -a reactors=("push" "cat")
declare -a connections=(1 2 4)
declare -a tx_sizes=(256 512 1024 2048)
LAST_REACTOR="${reactors[-1]}"

# Run experiments
for r in "${reactors[@]}"; do
    ORIGINAL_MANIFEST="networks/${NUM_NODES}nodes_${r}.toml"
    
    for c in "${connections[@]}"; do

        for s in "${tx_sizes[@]}"; do
            IX="n${NUM_NODES}_${r}_c${c}_s${s}"
            
            # Create manifest for this experiment
            MANIFEST="$MANIFESTS_DIR/m_${IX}.toml"
            cp $ORIGINAL_MANIFEST $MANIFEST
            sed -i'' -e "s/load_tx_connections = 1/load_tx_connections = $c/g" $MANIFEST
            sed -i'' -e "s/load_tx_size_bytes = 1024/load_tx_size_bytes = $s/g" $MANIFEST

            # Run experiment
            echo "🟢 reactor=${r}, c=${c}, s=${s} start="`date`
            ./build/runner -f $MANIFEST start
            sleep 10 # wait until nodes are running and stabilazed
            ./build/runner -f $MANIFEST load > /dev/null &
            sleep 240
            ./build/runner -f $MANIFEST logs > $LOGS_DIR/logs_${IX}
            ./build/runner -f $MANIFEST stop
            echo "🔴 reactor=${r}, c=${c}, end="`date`
            rm -rdf "$MANIFESTS_DIR/m_${IX}"

            if [ $s -ne ${tx_sizes[-1]} ]; then
                sleep 60
            fi
        done

        if [ $c -ne ${connections[-1]} ]; then
            sleep 60
        fi
    done

    if [ "$r" != "$LAST_REACTOR" ]; then
        sleep 60
    fi
done

rm -rdf $MANIFESTS_DIR

# Stop Prometheus
# kill -9 $(ps aux | grep '[p]rometheus' | awk '{print $2}')
