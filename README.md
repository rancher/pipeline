# Quick Start Guide

In this guide, you'll learn how to deploy the Rancher Pipeline service and create a pipeline for a simple nodeJS service as an example.

## Prerequisites

* **A Rancher environment** &mdash; To run Rancher Pipeline, we recommend more than two cores and 4GB memory for each host.

* **A GitHub account** &mdash; Fork a repository from this [pipeline-example repository](https://github.com/biblesyme/pipeline-example). 

## Deploying Rancher Pipeline

Rancher Pipeline is available in the [community-catalog](https://github.com/rancher/community-catalog). 

### To Deploy Rancher Pipeline:

1. On the Rancher UI menu, click Catalog **Catalog**. The Catalog page displays.
2. Search for the **Rancher Pipeline** template, and then click **View Details**. Configuration options for the template display.
2. Ignore the configuration options for now, and click **Launch** to deploy the Pipeline stack. This process might take a few minutes. All components of your stack begin running, and a new Pipeline tab displays on the UI menu.

When all services in the Pipeline stack are in an `Active` state, click **Pipeline** to access Pipeline UI.

## Configuring Source Code Management Authentication

Before adding a pipeline, first we need to configure source code management authentication. We will use Github OAuth for authentication in this Quick Start Guide.

### To Configure Source Code Management Authentication:

1. In the top-right corner of the Pipeline UI, click the gear icon. The Settings page displays.

2. In the **Git Authentication** section, follow the instructions to complete the authentication. Once complete, your GitHub username  displays in the **Authenticated Users** section.

## Adding a Pipeline

Now we're going to add a pipeline and set up a source code management stage, a test stage to run a 'mock' test and build CI jobs, and a package stage for it. 

### To Add a Pipeline:

1. In the Pipeline UI, click **Pipelines**.
2. Click **Add Pipeline** to create a new pipeline.

### To Set up a Source Code Management Stage:

1. To configure the source code repository for the pipeline, select the user and the forked repository.
2. Click **Add**.

### To Add a Test Stage: 

1. On the pipeline graph, click **Add a Stage**. Name it `test_stage` and click **Add**. 
2. Click **Add a Step** under the `test_stage`. 
3. Select **Run as a Service** because we need to test this server in the next section. 
4. For **Name**, type `nodeserver`. We'll use this name in the next section.
5. For **Image**, select `node:8`. We're using this Docker image to run our server container.
6. For **Command**, type:
```
npm start
```
7. Click **Save**.

Next, we need to add another step in `test_stage` to test our server. 

### To Test Our Server:

1. Click **Add a Step** under the `test_stage`. 
2. For **Image**, select `node:8`.
3. For **Command**, type:
```
npm install
npm test
```
4. Under **Environment Variables**, click **Add Variable**. 
5. For **Variable**, type `SERVER`. For **Value**, type `nodeserver`. The testing code requires you to set the environment variable `SERVER` as the server address it will test against.
6. Click **Save**.

Lastly, we need to package our newly created server to a Docker image.

### To Add a Package Stage:

1. Click **Add a Stage** and name it `package_stage`. 
2. Click **Add a Step** under the `package_stage`.
3. For **Step Type**, select `build`. We're going to build and push an image. 
4. For **Image Tag**, enter `<your dockerhub-id>/pipeline-helloword:latest`.
5. Select **Push**. 
   >**Note:** If you haven't registered in the Rancher Registry setting, a notification for registry authentication displays. Before proceeding to the next step, follow the instructions to complete the authentication. Then, double-click **Push** to refresh and confirm that your registry is ready. 
6. Click **Save**.

Now we've set up a source code management stage, a test stage, and a package stage for the pipeline. Give a name to the pipeline, such as `hello-world`, and then click **Save**. Your `hello-world` pipeline displays in the list on the Pipeline page.

## Running a Pipeline

To manually run the pipeline, from the pipeline actions drop-down list on the right, click **run**.

By default, we've also registered a Webhook in GitHub. When you push to the GitHub repository, it will trigger a run of the associated pipeline. This Webhook only works if your Rancher Server is accessible from GitHub.

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

# Contact

For bugs, questions, comments, corrections, suggestions, etc., open an issue in
[rancher/rancher](//github.com/rancher/rancher/issues) with labels `area/pipeline, version/1.6`.

Or just [click here](//github.com/rancher/rancher/issues/new?title=%5Bpipeline%5D%20&labels=area%2Fpipeline,version%2F1.6) to create a new issue.

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
Please refer to [Reference Guide](./docs/README.md) for more information.

