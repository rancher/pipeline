# CICD Deploy Guide on Rancher 1.6.x

***Note: If you need a access point of pipeline-ui in 1.6.x. Then `rancher/server:v1.6.10` is recommended, and follow the steps in [Access point in UI 1.6.10](#access-point-in-ui-1.6.10)***

We should be able to deploy cicd via one button click using default library catalog when everything is ready.



Remaining dependencies:

1. Related webhook-service update is not merged.
2. CICD UI code is not merged.



To deploy & test CICD:

1. Set up rancher/server:1.6.8+（To support updatable Generic objects） and add hosts.

2. Substitute webhook-service from 
```
https://github.com/biblesyme/webhook-service, branch: service-webhook
```
(git pull, make and get `webhook-service` binary )

built binary: `https://rancher.slack.com/files/U4E1KJKA7/F7RQZGYSY/webhook-service`

```
docker cp webhook-service <rancher-server-container>:/usr/local/bin/
docker exec <rancher-server-container> pkill webhook-service
```

3. Deploy CICD catalog

```
# Catalog, the name matters, currently UI use it to detect CICD services.
Name: CICD,
URL:https://github.com/gitlawr/rancher-catalog-1.git,
BRANCH:master
```

```
Item: "Rancher CICD",
Version: <latest version>,
# Default or customized configs and launch
```

4. After all services are ready, access CICD UI via `http://<rancher-server-ip>:8080/r/projects/1a5/pipeline-ui:8000/#/env/1a5/pipelines/r`(env id and server port is changable according to where you deploy cicd catalog)

## Access point in UI 1.6.10

### Prerequisites
* `rancher/server:v1.6.10` installed.

### Steps
1. Download https://rancher.slack.com/files/U2XC5VC8L/F7Z18V7U3/dist.zip

2. Unzip it.
3. Run 
> export UI=$(docker exec < rancher-server-container > find /usr/share/cattle/ -name index.html -maxdepth 2 | sed 's#/[^/]*$##' | sort -u)

> docker cp ${PWD}/dist/. < rancher-server-container >:${UI}