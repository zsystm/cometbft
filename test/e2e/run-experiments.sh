#!/usr/bin/env bash

set -ex

[ $# -eq 0 ] && echo "No arguments supplied: path to yaml file needed. For example: `./experiments/config-example.yml`" && exit 1
PARAMS_YAML=$1

# Parse yaml file and set parameters for experiment
manifest_template_path=$(yq .manifest_template_path $PARAMS_YAML)
duration=$(yq .duration $PARAMS_YAML)
interval=$(yq .interval $PARAMS_YAML)
IFS=',' read -r -a reactors <<< "$(yq ".reactors | @csv" $PARAMS_YAML)"
IFS=',' read -r -a connections <<< "$(yq ".load.connections | @csv" $PARAMS_YAML)"
IFS=',' read -r -a tx_rates <<< "$(yq ".load.tx_rates | @csv" $PARAMS_YAML)"
IFS=',' read -r -a tx_sizes <<< "$(yq ".load.tx_sizes | @csv" $PARAMS_YAML)"

# Directories and paths
OUTPUT_DIR=${PARAMS_YAML%.yml}/`date +"%FT%H%M%z"`
mkdir -p $OUTPUT_DIR
LOGS_DIR=$OUTPUT_DIR/logs/
mkdir -p $LOGS_DIR

MANIFEST_TMP_DIR="${manifest_template_path%.toml}"
MANIFEST_BASENAME=$(basename $manifest_template_path)

# Start Prometheus server (if it's not already running; we use the same instance for all runs)
./build/runner -f $manifest_template_path setup
PROMETHEUS_FILE="$OUTPUT_DIR/prometheus.yaml"
mv "$MANIFEST_TMP_DIR/prometheus.yaml" $PROMETHEUS_FILE
rm -rdf $MANIFEST_TMP_DIR
prometheus --config.file=$PROMETHEUS_FILE > /dev/null &
sleep 2

# Prepare
cp $PARAMS_YAML $OUTPUT_DIR
GRAFANA_ANNOTATIONS="$OUTPUT_DIR/grafana_annotations.csv"
echo "# start time, end time, label" >> $GRAFANA_ANNOTATIONS

# Run experiment
for reactor in "${reactors[@]}"; do

    for c in "${connections[@]}"; do

        for r in "${tx_rates[@]}"; do

            for s in "${tx_sizes[@]}"; do

                INSTANCE="${reactor}_c${c}_r${r}_s${s}"
                
                # Create manifest for this instance of the experiment
                MANIFEST="$OUTPUT_DIR/${MANIFEST_BASENAME%.toml}_${INSTANCE}.toml"
                cp $manifest_template_path $MANIFEST
                sed -i'' -e "s/mempool_reactor = .*/mempool_reactor = \"$reactor\"/" $MANIFEST
                sed -i'' -e "s/load_tx_connections = .*/load_tx_connections = $c/" $MANIFEST
                sed -i'' -e "s/load_tx_batch_size = .*/load_tx_batch_size = $r/" $MANIFEST
                sed -i'' -e "s/load_tx_size_bytes = .*/load_tx_size_bytes = $s/" $MANIFEST

                # Run instance
                echo "🟢 $INSTANCE, start="`date`
                START_EPOCH=$(date +"%s")
                ./build/runner -f $MANIFEST start
                sleep 10 # wait until nodes are running and stabilazed
                ./build/runner -f $MANIFEST load 1> /dev/null &
                sleep $duration
                ./build/runner -f $MANIFEST logs > $LOGS_DIR/logs_${INSTANCE}
                ./build/runner -f $MANIFEST stop
                ./build/runner -f $MANIFEST cleanup
                echo "🔴 $INSTANCE, end="`date`
                echo "$START_EPOCH, $(date +"%s"), $INSTANCE" >> $GRAFANA_ANNOTATIONS
                # rm -rdf "$MANIFESTS_DIR/m_${INSTANCE}"

                if [ $s -ne ${tx_sizes[-1]} ]; then sleep $interval; fi
            done

            if [ $r -ne ${tx_rates[-1]} ]; then sleep $interval; fi
        done

        if [ $c -ne ${connections[-1]} ]; then sleep $interval; fi
    done

    if [ "$reactor" != ${reactors[-1]} ]; then sleep $interval; fi
done

# Stop Prometheus
# kill -9 $(ps aux | grep '[p]rometheus' | awk '{print $2}')
