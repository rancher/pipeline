#!/bin/bash
# wait-for-master.sh

set -e

cmd="$@"

MASTER_URL=""
if [ ! -z "$JENKINS_MASTER" ]; then
  MASTER_URL="$JENKINS_MASTER/tcpSlaveAgentListener/"
else
  if [ ! -z "$JENKINS_SERVICE_PORT" ]; then
    MASTER_URL="http://$SERVICE_HOST:$JENKINS_SERVICE_PORT/tcpSlaveAgentListener/"
  fi
fi

while [[ $(curl -s -w "%{http_code}" $MASTER_URL -o /dev/null) != "200" ]]; do
  >&2 echo "master is unavailable - sleeping"
  sleep 5
done
>&2 echo "master is up - executing command"
exec $cmd
