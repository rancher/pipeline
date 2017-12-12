# CICD Deploy Guide on Rancher 1.6.x

Updated on `December 7

We should be able to deploy cicd via one button click using default community catalog when `pipeline  ` catalog item is merged.


To deploy & test CICD:

1. Set up rancher/server:v1.6-development（To support updatable Generic objects） and add hosts.

2. Deploy CICD catalog
```
Name: CICD,
URL:https://github.com/gitlawr/community-catalog.git,
BRANCH:cicd
```

```
Item: "Rancher Pipeline",
# Use default version
# Default or customized configs and launch
```

4. After the deployment, You can see a `Pipeline` tab showing up on the top-right navigation bar.

