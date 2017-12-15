# Quick Start Guide

In this guide, we will deploy the Rancher Pipeline service and create a pipeline for a simple nodeJS service as an example.

## Prerequisites

* A Rancher environment. For each host, it is recommended to have more than 2 cores and 4 GB memory for running Rancher Pipeline.

* A GitHub account which has a repository forked from this [pipeline-example repository](https://github.com/biblesyme/pipeline-example). 

## Deploy Rancher Pipeline

Rancher Pipeline is available in [community-catalog](https://github.com/rancher/community-catalog). 

1. Navigate to the **Catalog** tab, search **Rancher Pipeline** and click on Rancher Pipeline template. 
2. You will see configuration options for the template. For now, just simply click **Launch** to deploy the Pipeline stack, all components will be up shortly and a new tab named `PIPELINE` will appear in the top navigation bar.

When all services in pipeline stack are in the `Active` state, click on the **PIPELINE** tab on the top navigation bar to access Pipeline UI.


## Configure Source Code Management authentication

Before adding a pipeline, we need to configure source code management authentication first. We will use Github OAuth for authentication in this quick start guide.

1. Click on the gear icon in the top right corner of the Pipeline UI, which will redirect you to the setting page.

2. In the setting page, under the **Git Authentication** section, you need to follow the instructions to complete the authentication. You should be able to see your GitHub username in the **Authenticated Users** after the authentication has been completed correctly.

## Add a pipeline

1. Click on **Pipelines** tab in pipeline UI.
2. Click **Add Pipeline** button on the right to create a new pipeline.

First, you will need to configure the source code repository for the pipeline. Select the user and the forked repository, then click **Add** button.

### Add a Stage named `test_stage`
Next, we will add some more stages/steps to run 'mock' test & build CI jobs.
1. Click **Add a Stage** on the pipeline graph. Name it as `test_stage` and click **Add** button. 
2. Click **Add a Step** under the `test_stage`. 
3. Check the `Run as a Service` checkbox, cause we need to test this server in the next Step. 
4. Fill **Name** with `nodeserver`. We will use the name `nodeserver` in the next Step.
5. Fill **Image** with `node:8`. We will use it as the Docker image to run our server container.
6. Fill **Command** with
```
npm start
```
7. Click **Save** button.

We will need another Step in `test_stage` to test our server. 
1. Click **Add a Step** under the `test_stage`. 
2. Fill **Image** with `node:8`.
3. Fill **Command** with
```
npm install
npm test
```
4. Under **Environment Variables**, click `Add Variable`, fill the **Variable** with `SERVER` and **Value** with `nodeserver`. It's because the testing code requires environment variable `SERVER` to be set as the server address it will test against.
5. Click **Save** button.

### Add a Stage named `package_stage`
We will package our newly created server to a Docker image in this stage.
1. Click **Add a Stage** and add another stage named `package_stage`. 
2. Click **Add a Step** under the `package_stage`.
3. Select **Step Type** as `build`. We're going to build and push an image. 
4. Fill the **Image Tag** with `<your dockerhub-id>/pipeline-helloword:latest`.
5. Select the **push** option. You will see a notification for registry authentication if you haven't registered in the Rancher Registry setting. Follow it to complete the authentication. Then go back to the Step configuration.
6. Double click the **push** option to refresh and confirm that registry is ready. 
7. Then click **Save**.

Now we've set up a source code management stage, a test stage and a package stage for the pipeline. Give a name to the pipeline as `hello-world` in the left-top input, then click **Save** to save our pipeline. You should see the `hello-world` pipeline in the pipeline list page.

## Run a pipeline

To manually run the pipeline, click the dropdown of the pipeline actions on the right and click **run**.

By default, we've also registered a Webhook in GitHub. So when you make a push to the GitHub repository, it will trigger a run of the associated pipeline. Notice that it will only work if your Rancher Server is accessible from GitHub.

# Development

If you wanna make and test Rancher Pipeline yourself, just follow steps below.

## Build

1. Clone this repository.
2. Run `make` under your cloned directory.
3. After success of step 2, there will be four images(rancher/jenkins-slave:<-version->, rancher/pipeline:<-version->, rancher/pipeline-ui:<-version->, rancher/jenkins-boot:<-version->) built. Tag and Push them to your Registry.

## Set Up Test Environment

### Prerequisites

* A Rancher environment. For each host, it is recommended to have more than 2 cores and 4 GB memory for running Rancher Pipeline.

### Steps
There are two options you can select to set up the test environment.

#### Option 1
1. [Deploy Rancher Pipeline](#deploy-rancher-pipeline)

2. Upgrade every service under `pipeline` stack to the image you built.

#### Option 2
1. Fork [community-catalog](https://github.com/rancher/community-catalog) to your GitHub account.

2. Under your forked repository from step 1, modify images used in `infra-templates/pipeline/<-version->/docker-compose.yml.tpl` to the images you built.

3. Add this forked repository as a Catalog in your Rancher Environment.

4. [Deploy Rancher Pipeline](#deploy-rancher-pipeline) in the Catalog of Step 3

# License
Copyright (c) 2014-2017 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

# Reference Guide
Please refer to [Reference Guide](https://github.com/rancher/pipeline/blob/docs/docs/README.md) for more information.

