# CICD Deploy Guide on Rancher 1.6.x

We should be able to deploy cicd via one button click using default library catalog when everything is ready.



Remaining dependencies:

1. Related webhook-service update is not merged.
2. CICD UI code is not merged.



To deploy & test CICD:

1. Set up rancher/server:1.6.x and add hosts.

2. Substitute webhook-service from 
```
https://github.com/biblesyme/webhook-service, branch: service-webhook
```
(git pull, make and get `webhook-service` binary )
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
Item: "Rancher CICD Dev Version",
Version: "0.6.10",
# Default or customized configs and launch
```

4. After all services are ready, access CICD UI via `http://<rancher-server-ip>:8080/r/projects/1a5/pipeline-ui:8000/#/env/1a5/pipelines/r`(env id is changable according to where you deploy cicd catalog)
