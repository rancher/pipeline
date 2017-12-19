# Rancher Pipeline Reference Guide

Easy to use, Easy to integrate CI/CD with Rancher.

## Table of Contents

- [User Guide](#user-guide)
  - [Core Concepts](#core-concepts)
  - [Step Types](#step-types)
  - [Source Code Management Integration](#source-code-management-integration)
  - [Triggers](#triggers)
  - [Environment Variables](#environment-variables)
  - [Conditions](#conditions)
  - [Pipeline File](#pipeline-file)
- [Admin Guide](#admin-guide)
  - [Installation](#installation)
  - [Backup/Restore](#backuprestore)

## User Guide

## Core Concepts

This section provides some contexts and terminologies used throughout the documentation.

### What is Continuous Integration(CI)

Continuous Integration is the practice of merging all developer working copies to a shared [mainline](https://en.wikipedia.org/wiki/Trunk_(software)) frequently. The main aim of CI is to detect and track down integration bugs early and easily by submitting and testing small change sets, so as to avoid the pitfalls of "integration hell".

Rancher Pipeline helps you to define and run your CI processes. It clones your source code repository into a  fresh, isolated environment in containers, runs build and test tasks for your code and provides feedback on the result. 

### Pipelines, Stages and Steps

#### Pipeline

A pipeline is a construct defining a CI process. A pipeline consist of multiple stages, it starts with Source Code Management and goes with building, testing, and deployment. A pipeline can be configured via UI and it can also be viewed/imported/exported as a [pipeline file](#pipeline-file) so that it can be versioned and reviewed "as code". Each run of a pipeline generates a history record of the pipeline.

#### Stage

A `Stage` consists of a group of actions, known as `Steps`. Stages run sequentially. Steps in a stage can run in sequence or parallel, by selecting **Step Running Mode** in a stage configuration. When they run in parallel, the running order is not guaranteed and the number of concurrent steps is dependent on the number of executors in slave nodes.

You can configure `approvers` on a `Stage` so that a pipeline execution will be pending when it reaches this stage. After `approver` approves this stage of the pipeline, it will continue to run. 

You can configure [**conditions**](#conditions) for when to run a stage.

> **Note:** you can't configure name, approvers or conditions on the first stage because it is designed to be an initial stage to checkout source code.

#### Step

A `Step` is a minimum execution unit, and fundamentally it describes what to do. For example, to run a shell command or to build a Docker image. There are different types of `Step` dedicated to different tasks. For more details, see [Step Types](#step-types)

There are some common configurations for all kinds of steps:

You can configure **Timeout** of a single step in minutes. When a step does not complete by the specified amount of time, the step will fail.

You can configure [**Conditions**](#conditions) for when to run a step.

## Step Types

There are several built-in types of step:

### Source Code Management(SCM)

Source Code Management step is designed to be the first step of a pipeline. When you add a new pipeline, you are required to set up the Souce Code Management step.


In SCM step, Choose an authenticated account( for how to do authorization please refer to [Source Code Management Integration](#source-code-management-integration)) and select a repository, fill the **Branch** field to specify which git branch to use.

If you enable the **Webhook** option, it means that the pipeline can be triggered by a push event webhook. For example, if you are using Github here, a Github webhook will be automatically created and once a push happens in relevant repository & branch, Github will send a POST request to Rancher server and trigger a new run of the pipeline.

### Build

Build step is for building a docker image. There are normally two cases of building an image in continuous integration using Docker. 

1. Build a context image to be used in following tasks such as testing.
2. Build an image as the artifact of the CI process, which can be used in later testing and deployment.

You can upload a Dockerfile or use an existing Dockerfile in your source code repository. You can configure the **Build Path** and **Image Tag** that you are going to build. The image tag should be the full image name containing image registry prefix.

To push the built image to a registry, simply click and enable the **push** option. Rancher Pipeline uses [registry credentials](http://rancher.com/docs/rancher/latest/en/environments/registries/) which are stored in Rancher server. If related registry credential is not configured yet, The UI will notice and guide you there.

### Task

Task step is for running arbitrary shell commands in an arbitrary context image. For example, running PHPUnit tests using the php image or running a `go test` using the golang image. You can specify a context image in **Image** field. To use previously referenced images in the pipeline, you can click drop-down button on the right and select one of them.

When you input some commands in the **Command** textarea, these commands will be wrapped in a file and executed by `/bin/sh`. To run with a custom entrypoint and commands, leave **Shell Script Command** empty, then you can click **Custom Entrypoint** and configure them according to your need.

There is also an option called **Run As a Service** in task step. When it is enabled, it means that the task step is meant to be a long-running container during the lifecycle of the pipeline execution. It can be referenced by later steps using its alias which is specified in the **Name** field. For example, if you configure a mysql task step which runs as a service with **Name** `mysqltest`, you can connect to the mysql database using `mysqltest` as the host. This option can be useful when your tests depend on middleware services such as databases.

> Note: 
>
> 1. Rancher Pipeline does not do health check for these services so users are responsible for ensuring that they are up and ready.
> 2. All running services will be cleaned up when a pipeline execution is finished.

### Upgrade Service

Upgrade Service step is for upgrading docker image for [Rancher services](http://rancher.com/docs/rancher/latest/en/cattle/adding-services/#services). To select the group of services to be upgraded, you would use a or multiple selector labels that will pick up any service that contains the matching labels. Matching services will be upgraded to use the image which is configured in the step. Labels should be added to a service when creating the service. If the label doesn’t exist, you will need to upgrade the service in Rancher to add the label to the upgrade service step.

In order to upgrade the services, Configure the following:

- Set the image to upgrade.


- Select the label to find the services to be upgraded
- Determine the number of containers to be upgraded at a time (i.e. Batch Size)
- Determine the number of seconds between starting the next container during upgrade (i. e. Batch Interval)
- Select whether or not the new container should start before the old container was stopped

By default, Rancher Pipeline searches and upgrades matching services in current environment, to upgrade services in another environment, click **Target another environment** and fill in [environment API keys](http://rancher.com/docs/rancher/latest/en/api/v2-beta/api-keys/#environment-api-keys) for that environment.

### Upgrade Stack

Upgrade Stack step is for upgrading the whole definition of a [Rancher stack](http://rancher.com/docs/rancher/latest/en/cattle/stacks/). Configure **Stack Name** to the name of the stack that you want to upgrade. Rancher [stack configuration](http://rancher.com/docs/rancher/latest/en/cattle/stacks/#stack-configuration) is defined by `docker-compose.yml` and `rancher-compose.yml`, you can fill in the compose file fields for the upgrade. For convenience, you can input an increment part of the compose files for the upgrade, For example, to upgrade an `ngx` service with new image tag, you can simply input following texts in **docker-compose.yml** textarea of upgrade stack step:

```
services:
  ngx:
    image: nginx:<new-tag>
```

Then Rancher Pipeline will merge the above configuration into original stack docker-compose definition and do the upgrade. Note that the path from `image` key to the root of the compose file is needed here.

If you want to remove a field in the original configuration, you can override that key with the empty value.

By default, Rancher Pipeline searches and upgrades matching services in current environment, to upgrade services in another environment, click **Target another environment** and fill in [environment API keys](http://rancher.com/docs/rancher/latest/en/api/v2-beta/api-keys/#environment-api-keys) for that environment.

### Upgrade Catalog

Upgrade Catalog step is for upgrading a Rancher [catalog](http://rancher.com/docs/rancher/latest/en/catalog/#catalog) template, it helps to tag a new catalog template version and push it to your catalog repository.

To do the upgrade, select a catalog from the catalog list. You can click **Edit** button and add a new one. Then select the app from **Select App** dropdown, which is the catalog template you want to upgrade and tag.

Then you can input new  `README.md`, `docker-compose.yml`, `rancher-compose.yml` templates for it. You can also click **Select Template From Old Version** button, choose a version then click the **Select** button so that you can edit new temples files based on original definitions.

You can also choose to upgrade a stack of this catalog template to the latest version by enabling **Upgrade to the latest version** option.

## Source Code Management Integration

Pipelines start with source code management step. Before adding and running a pipeline, you are required to add source code management authentication. Rancher pipeline has built-in support for following source code management tools, you can configure them at runtime and enable multiple kinds at the same time.

To configure source code management integration, click on the gear icon on top-right for the setting page.

> Note: Pipelines and authenticated accounts are shared between users in the environment. They can configure and run the pipelines with these accounts. So it is recommended to use a shared Git account to run CI jobs or use a dedicated environment and give access to authorized users.

### Github

Rancher Pipeline uses Github OAuth to do authentication. You can following the guide in setting page to do the authentication. You can use public Github service or use a private Github enterprise installation.

After doing following steps:

1. Set up a Github application
2. configure to use your application
3. click **authenticate with github**

Your browser will prompt a window for your login and authorization on Github. After these are done, the application is configured and your account is authenticated to Rancher Pipeline.

Multiple Github accounts can be added in the Git authentication settings. To add more accounts, click **Authenticate with Github** button. Note that everytime the authentication will ask for authorization of current Github user. In order to add another Github account, you may need to log out on Github first.

### GitLab

Rancher Pipeline uses GitLab OAuth to do authentication, in a similar way as Github. You can following the guide in setting page to do the authentication. You can use public GitLab service or use a private GitLab installation.

After doing following steps:

1. Set up a GitLab application
2. configure to use your application
3. click **authenticate with gitlab**

Your browser will prompt a window for your login and authorization on GitLab. After these are done, the application is configured and your account is authenticated to Rancher Pipeline.

Multiple GitLab accounts can be added in the Git authentication settings. To add more accounts, click **Authenticate with gitlab** button. Note that everytime the authentication will ask for authorization of current GitLab user. In order to add another GitLab account, you may need to log out on Github first.

## Triggers

There are multiple ways to trigger a pipeline to run. To disable automatic triggers including webhook and cron, you can deactivate a pipeline by clicking **deactivate** in action drop-down, or disable **active** option on pipeline editing page.

### Manual Trigger

You can manually trigger a pipeline to run, by clicking **run** in the action drop-down of a pipeline.

### Webhook Trigger

When the **webhook** option in source code management step is enabled, Rancher pipeline will automatically create a project webhook in related source control server, with a generated token for webhook validation. A pipeline execution can be triggered by webhook with following conditions satisfied:

1. The pipeline is in `active` status.
2. The **webhook** option in source code management step is enabled.
3. Rancher server is available to receive webhooks from Github, GitLab, etc.

### Cron Trigger

In pipeline editing page, you can configure cron trigger in **Schedule** tab.

You can input a [cron expression](https://en.wikipedia.org/wiki/Cron) in **Internal Pattern** field, which is made of five fields. You can select a **Cron Timezone**, which is by default your detected local timezone.

There is an option **Run when there is new commit**. When it is enabled, everytime a cron schedule is carried out, Rancher Pipeline will see if there is any new commit in the branch of the repository since the last run of the pipeline. A new run of the pipeline is triggered only when new commits are there.

## Environment Variables

Environment variables can be used in both pipeline configurations and shell script runtime environment. When you input '$' in pipeline configuration inputs, we will pop up available variables for you to choose. There are following kinds of environment variables:

#### Pre-define variables

The following variables are available in most inputs, including shell scripts, image tag, compose file template, etc. (except `git branch` and `timeout` config currently). 

| NAME                   | DESC                                  |
| ---------------------- | ------------------------------------- |
| CICD_GIT_COMMIT        | git commit sha                        |
| CICD_GIT_BRANCH        | git branch                            |
| CICD_GIT_URL           | git repository url                    |
| CICD_PIPELINE_ID       | pipeline id                           |
| CICD_PIPELINE_NAME     | pipeline name                         |
| CICD_TRIGGER_TYPE      | trigger type                          |
| CICD_NODE_NAME         | jenkins node name                     |
| CICD_ACTIVITY_ID       | pipeline history record id            |
| CICD_ACTIVITY_SEQUENCE | run number of pipeline history record |

#### User-defined variables

Users can add user-defined parameters in pipeline configuration(**Parameters** configuration on Pipeline editing page). They act as the same role except that they are defined by users.

#### Environment variables in task steps

You can configure environment variables for a task step. Unlike pre-define or user-defined variables that work in the whole pipeline configuration, these environment variables are limited to the container context running the task. Therefore they are not available in some configurations such as **image** of this step or in configurations of other steps. 

Environment variables in step configuration take precedence over global variables when they are overlapped.

## Conditions

You can specify conditions of running a step/stage. When conditions are added, they will be checked before running a step/stage. If the conditions are met, the step/stage runs as usual. If the conditions are not met, the step/stage is skipped and following steps/stages continue.

Conditions consist of expressions, each in the form `<envvar> <operator> <value>`. Pre-define or user-defined variables are supported here. `=` for `equal to` and `!=` for `not equal to ` are supported as the operator. You can combine multiple expressions and choose to run the step/stage when all/any of the expressions are true.

## Pipeline File

Pipeline definition is not required to be stored in source code repository, but you can view/export/import a pipeline as a pipeline file. This can be useful for the continuous integration workflow to be versioned, reviewed and migrated to different deployment.

To view the pipeline file of a pipeline, click **View Config** in action drop-down of a pipeline.

To export the pipeline file of a pipeline, click **Export Config** in action drop-down of a pipeline.

To Import a pipeline file, click **Import pipeline.yml** button in pipeline list page.

### Pipeline File Reference

```
# pipelinefile.yaml
version: v1
# pipeline name
name: <string>
# enable/disable automatic triggers
isActive: <bool> 
parameters: []<string> # In `key=val` format
#cron trigger keys
cronTrigger:
  triggerOnUpdate: <bool> # trigger when there's new commit
  spec: <string> # cron expression
  timezone: <string> # cron trigger timezone

stages: #array
  - Name: <string>
    needApprove: <bool>
    parallel: <bool>
    approvers: ["id1","id2"] #<sting[]> for user ids
    # either all or any is used, each condition should be in `ENVVAR=VAL` or `ENVVAR!=VAL` format.
    conditions:
      all: <[]string>
        - "CICD_GIT_BRANCH=master"
        - "CICD_GIT_BRANCH!=master"
      any: <[]string>
        steps: [<step_spec>]


# <step_spec>:
# generic keys
#enum{"scm","task","build","upgradeService","upgradeStack","upgradeCatalog"}
type: <string>
conditions:
  # either all or any is used, each condition should be in `ENVVAR=VAL` or `ENVVAR!=VAL` format.
  all: <[]string>
    - "CICD_GIT_BRANCH=master"
    - "CICD_GIT_BRANCH!=master"
  any: <[]string>


#--- for `scm` type
scmType: <string> #enum{"github"},takes no effect currently
repository: <string>
branch: <string>
gitUser: <string> # In the form of <sourceType>:<username>
webhook: <bool> #whether or not generates webhook automatically


#--- for `build` type
dockerFileContent: <string> # dockerfile content, if not set, use default dockerfile path for docker build.
dockerFilePath: <string> # dockerfile path, ignore if `dockerFileContent` is set.
buildPath: <string> # docker build path, using "." as default.
targetImage: <string> # image name to build
push: <bool> # whether push the built image or not


#--- for `task` type
image: <string> # context image to run the task
isService: <bool> # whether run "as a service" or not
alias: <string> # alias to be referenced by other steps. ignore when `isService==false`
env: []<string> # environment variables of task step, in `key=val` format.


shellScript: <string> # shell script to run, will wrap it in a shell script file and run /bin/sh as the entrypoint.
entrypoint: <string> # entrypoint to run, will ignore if `shellScript` is not empty
args: <string> # the command for docker run, will ignore if `shellScript` is not empty


#--- for `upgradeService` type
imageTag: <string> # image for the services to upgrade (maybe change to "image"?)
serviceSelector: <map> # service selector
batchSize: <int>
interval: <int>
startFirst: <bool>
endpoint: <string> # rancher server api endpoint when deploy to other env. If endpoint&api keys is not set, will deploy to current environment by default.
accesskey: <string> # rancher server api key to use when deploy to other env.
secretkey: <string> # rancher server api key to use when deploy to other env. This key Will not be exported so you may need to fill in the key when import a pipeline


#--- for `upgradeStack` type
stackName: <string> # stack name to upgrade
dockerCompose: <string> # docker compose file content for the stack to upgrade
rancherCompose: <string> # rancher compose file content for the stack to upgrade
endpoint: <string> # rancher server API endpoint when deploying to other environments. If endpoint&api keys are not set, will deploy to current environment by default.
accesskey: <string> # rancher server API key to use when deploying to other environments.
secretkey: <string> # rancher server API key to use when deploying to other environments. This key Will not be exported so you may need to fill in the key when importing a pipeline


#--- for `upgradeCatalog` type
externalId: <string> # externalId for the catalog. (seems not user friendly,split it?)
templates: # <map> for docker-compose,rancher-compose,readme files. 
  <string>: <string> # file-name:file-content
deploy: <bool> # whether deploy catalog stack to latest upgraded catalog version or not
stackName: <string> # stack name to upgrade to latest catalog version, ignore when `deploy==false`
answers: <string> # answer file content to deploy the latest catalog, ignore when `deploy==false`
endpoint: <string> # rancher server API endpoint when deploying to other environments. If endpoint&api keys are not set, will deploy to current environment by default.
accesskey: <string> # rancher server API key to use when deploying to other environments.
secretkey: <string> # rancher server API key to use when deploying to other environments. This key Will not be exported so you may need to fill in the key when importing a pipeline

```

## Admin Guide 

## Installation

Rancher Pipeline is available in the [community-catalog](https://github.com/rancher/community-catalog).

To Deploy Rancher Pipeline:

1. Prepare a Rancher environment. To run Rancher Pipeline, we recommend more than two cores and 4GB memory for each host.
2. On the Rancher UI menu, click Catalog **Catalog**. The Catalog page displays.
3. Search for the **Rancher Pipeline** template, and then click **View Details**. Configuration options for the template display.
4. Fill in the configuration options, and click **Launch** to deploy the Pipeline stack. This process might take a few minutes. All components of your stack begin running, and a new Pipeline tab displays on the UI menu.

When all services in the Pipeline stack are in an `Active` state, click **Pipeline** to access Pipeline UI.

### Pipeline Catalog Configuration options

- **# of slaves**: The number of Jenkins slave to set up, one by default. Please set at least one slave. You can also do scaling of slaves after installation.
- **# of executors**:  The number of executors on each Jenkins slave. The maximum number of concurrent builds that Jenkins may perform on an agent. A good value to start with would be the number of CPU cores on the machine. Setting a higher value would cause each build to take longer, but could increase the overall throughput. For example, one build might be CPU-bound, while a second build running at the same time might be I/O-bound — so the second build could take advantage of the spare I/O capacity at that moment. Agents must have at least one executor.
- **Host with Label to put pipeline components on**: This parameter specifies the host labels to use. Pipeline components will be scheduled to dedicated hosts matching these host labels.

>Note: Pipeline steps are mapped to Jenkins jobs, and they are assigned to the slaves to be executed. Steps in a single run of a pipeline will be assigned to the same slave node to share the workspace.

## Backup/Restore

The Pipeline data are stored in two separate places, the pipeline definition and basic pipeline history status information are stored in Rancher server database, the detailed console log of pipeline history record is stored in Jenkins master volume. 

For pipeline data, it goes with Rancher server, so you don't need extra effort to backup/restore.

For detail console log, you can dump the Jenkins home directory in standard location of Jenkins master to your backup location.

## Clear Data

Pipeline data is persisted in Rancher server and it remains even if you remove the Rancher Pipeline deployment. If you want to clear related data, you can go to setting page and click **Clear Data**. Note that this is an unrecoverable operation.