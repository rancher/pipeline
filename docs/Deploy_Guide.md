# CICD Deploy Guide

We should be able to deploy cicd via one button click using default library catalog when everything is ready.



Remaining dependencies:

1. `volumes_from` in compose is not implemented.
2. Related webhook-service update is not merged.
3. CICD UI code is not merged.



Workaround steps to deploy & test CICD:

1. Run rancher/server:master with specific webhook-service branch:

```
docker run -d --restart=unless-stopped -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock -e REPOS="https://github.com/biblesyme/webhook-service,origin/service_webhook" rancher/server:master
```

2. \# Add host, **switch to `System` environment**, add CICD catalog manually.

```
# Catalog, the name matters, currently UI use it to detect CICD services.
Name: CICD,
URL:https://github.com/gitlawr/rancher-catalog-1.git,
BRANCH:master
```

3. Prepare preconfigured jenkins directory

```
CID=$(docker run -d reg.cnrancher.com/pipeline/jenkins_home:2.0.6.2_3 sleep 1)
docker cp $CID:/var/rancher/jenkins_home /var/jenkins_home
docker rm $CID
```

4. Deploy CICD catalog:

```
Item: "Rancher CICD Dev Version",
Version: "0.6.0-r2.0-v3-master",
# Default or customized configs and launch
```

5. Deploy UI service:

```
# you can add a scale-1 service to Default stack
# you can map to another port
image:reg.cnrancher.com/pipeline/ui:v3.1
port: 8083:8000
Env: Rancher=<http://IP:Port/>
```

6. After UI service is up, access it via <IP:8083>, swith to `system` environment, you should be able to see `pipeline` tab on the top bar.
