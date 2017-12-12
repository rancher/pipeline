# Design

## Overview

Rancher CICD is a sub system in Rancher environment for running continuous integration and continuous deployment tasks. It should be pluggable and easy to install in existing Rancher systems, via a catalog item. It is natively integrated with Rancher UI for consistent user experience and helps users to do CICD work in Rancher with ease.

## Architecture

The CICD system consists of three parts, Rancher server, Pipeline server and Jenkins, as is shown in following image:

![architecture](./images/architecture.png)

### `Rancher Server`

The CICD system is dependent on Rancher server and leverage its functionalities. Rancher server keeps persistence of CICD job configurations( known as `Pipelines`) and execution records (known as `activities`). These data is stored as generic objects in Rancher server. Rancher server also works as the gateway to receieve user requests and github webhooks and proxy them to Pipeline server.

### `Pipeline Server`

Pipeline server is the API server in CICD system. It provides API to handle Pipeline CRUD operations and other actions such as run, activate, etc. It does not do the actual CICD tasks but distribute them to Jenkins. Pipeline server is in charge of mapping a pipeline definition to Jenkins jobs, scheduling and running Jenkins jobs according to various CICD workflow, such as parallel or conditional tasks.

### `Jenkins`

Jenkins is the executor of CICD tasks. Jenkins jobs are configured and triggered by Pipeline server. Jenkins does what pipeline server ask and it will call back pipeline server on specific events(such as jobs start/end) to update activity status. Jenkins also stores detail logs of the activities and workspace contents in file system, which is mounted in local docker volumes. You can add arbitrary Jenkins slave by setting CICD catalog configuration according to your workload needs. Rancher cli and a dedicated tool running on Jenkins Jobs can directly talk to Rancher Server to do specific tasks, such as getting registry credential for pushing images, and deploying to Rancher environments.

Docker socket is bind mounted on slaves so slave nodes can leverage docker functionalities from docker on the host. However it is not exposed in the container that runs user-custom tasks. So the docker socket is not directly exposed to the users.

## Pipeline

Pipelines are the construct defining a CI flow. A Pipeline consists of stages and a stage consists of steps. Steps are minimum execution units that do the CI actions. Steps in the same stage can run in sequence or parallel while stages run in sequence. An execution of a pipeline is called an activity. An activity contains its pipeline model and the runtime status of the execution. Pipelines are expected to start with a SCM step.



## Mapping

Each step maps to a Jenkins job so that pipeline server can conveniently control the CI workflow. All steps of  an activity are assigned to the same Jenkins node so that they share the same workspace. Different types of steps map to Jenkins job configurations. A SCM step maps to Jenkins job source code management settings. A task step maps to `docker run` command in bash shell. A build step maps to `docker build` command in bash shell. An upgradeService/Stack/Catalog maps to a combination of commands that talk to Rancher server and do relevant updates.



## Storage

Pipeline/Activity data is majorly stored using Generic Objects in Rancher server, except for the detail logs of the activities, which are stored in Jenkins. We store Jenkins data and workspaces in local docker volumes on hosts running Jenkins master/slave.



## High Availability

High availability is currently not supported. Not easy to do with open source version Jenkins.

## Disaster Recovery

Pipeline server is stateless but when it is down, it fails to receive notifications from Jenkins events. When Pipeline server recovers it will try to sync status of running activities.

When Jenkins master is down, it can be recovered from Jenkins data.

## Detail Workflow

### 1. Setting up CICD catalog item

CICD consists of `jenkins_home`,`jenkins_master`,`jenkins_slave`,`pipeline_server` containers.

Jenkins_home contains a pre-configured jenkins directory. When it starts it mounts a volume and copies the pre-configured jenkins directory to the volume if it is empty.

Jenkins_master depends_on jenkins_home and inherit the pre-configured volume therefore Jenkins will be ready without interactions from users.

jenkins_slave is an optional part. Each slave will connect to master and act as an worker node.

Pipeline_server depends_on jenkins_master. It will serve after Jenkins master gets ready. Pipeline_server service is named as `pipeline-server`. UI will detect whether this service is ready and will show up after that.

### 2. Pipeline/Activity CRUD

Pipeline_server runs with `environment` role thus it has API keys for its environment. When pipeline/activity CRUD API is invoked, it will connect Rancher server and operate generic objects as pipeline data storage. Pipelines/activities are serialized in Json format and stored as key-value pairs.

### 3. Running a pipeline

When a run is triggered, the pipeline definition is mapped to Jenkins jobs. Each step is represented by a Jenkins freestyle job. The task of a Step is implemented using shell scripts. Pipeline server initially trigger the first step to run. everytime a step starts or finishes, Jenkins will notify pipeline server to update activity status. When a fail event arrives, Pipeline server marks the activity as failed. When a success event arrives, Pipeline server then trigger next step/stage or pend for approval or finish according to Pipeline configurations including conditions, parallel behavior, approval settings etc.

Database transaction is not directly used so when update an activity status a mutex lock is used for each activity to avoid concurrency conflicts.

### 4. Executing steps

All steps in an activity is expected to run in a shared workspace, which is the root directory of the source code. For that purpose,  When a pipeline runs, Pipeline server randomly choose an active Jenkins node for an activity, all of the steps are assigned to the selected node and related Jenkins jobs are configured to use the same workspace.

To run continuous integration using docker containers, the CICD system provides different kinds of steps and leverage docker commands for different CI phases. 

For pulling source code, we provide a `SCM` type step. It is mapped to Source code management configuration of Jenkins job. We provide two automated triggers to run pipelines, both can be related to source code updates, as described later in `webhook` and `cron trigger`

For generic tasks including building and testing codes we provide a `task` type step which is in fact a `docker run`. Users can choose a specific context image and run arbitrary commands within it. The shared workspace is mounted in the container context.

Docker images are the artifacts in CI using docker. To generate and collect the artifacts we provide a `build` type step which is in fact a `docker build & docker push`. Users can build a docker image using a Dockerfile in source code or that provided via UI. Users can choose to push the image to a registry after it is built. Rancher registry credentials are used here so users do not need to provide them seperately.

For continuous deployment we provide different step type to interact with Rancher server,including `upgradeService`,`upgradeStack`,`upgradeCatalog`. Tools including `Rancher cli` and `cihelper` is packed in Jenkins images and will be run in these steps. They can talk to Rancher server to do the deployment work, using API keys from `environmentAdmin` role.

### 5. Source Code Management Auth

SCM Authorization is useful for: webhook automated operations, pull/push accessibility for upgradeCatalog or others. Currently Github Oauth is supported. It can be done following guides that is similar to Rancher Github Auth. After authorization, CICD gets the user token and also stores it in generic objects.

### 6. Webhook 

At the start of pipeline server, it will check or create a `service_webhook` type Rancher webhook. It is used to receieve external systems' notifications, such as github webhooks without Rancher credentials. The webhook is a singleton and it can proxy to pipeline server and trigger all pipelines to run according to requests payload and headers. When a pipeline is created and `webhook` option is set, pipeline server will create a github webhook directing to this Rancher webhook. It containers pipelineId, environmentId, secret token generated for each pipeline, and is triggered whenever a new push event occurs. Pipeline server will validate the request payload and the secret token then run the pipeline if it is valid. Currently only Github webhook is available, integration of other system like Gitlab can be done in similar way.

### 7. Cron Trigger

We provide cron scheduler settings for a pipeline. There will be go routines running for the cron task if it is set. We provide a `run when new commits exist` option, it uses `git ls-remote` to check if there is any updates comparing to last run of the pipeline, and only runs a pipeline when there are new changes.

### 8. Multiple Git Account Support

1     A github account is bound to aRancher user. By default, an account is private and only accessible by itsowner Rancher user. We can set an account from `private` to `shared` then it can be accessible by anyone in this environment.

2     Users can only see and operate pipelines/activities of accessible github accounts. (including those they own and those shared by others)

3     If Access control is disabled.Accounts are shared over the environment.

4     When adding a pipeline, Users select an account first,then select a repo

### 9. Git Repo cache

Git repositories are cached so that users don't need to wait to load all repos from Git server. Respectly, users need to refresh git repositories manually if there are repo updates.

