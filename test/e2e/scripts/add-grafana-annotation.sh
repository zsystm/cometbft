#!/usr/bin/env bash
set -x

DASHBOARD_UID=$1
START_TIME=$2
END_TIME=$3
LABEL=$4

ADMIN_USER=admin
ADMIN_PASSWORD=admin
GRAFANA_URL=localhost:3000
# PANEL_ID=21 # Send bytes per message
PANEL_ID=9 # Peers

DATA='{
    "dashboardUID": "'$DASHBOARD_UID'", 
    "panelId": '$PANEL_ID',
    "time": '$START_TIME',
    "timeEnd": '$END_TIME', 
    "text": "'$LABEL'",
    "tags": ["'$LABEL'"]
}';
curl -s -k -XPOST -H 'Content-Type: application/json' --data "$DATA" \
    http://$ADMIN_USER:$ADMIN_PASSWORD@$GRAFANA_URL/api/annotations | \
    jq .message
