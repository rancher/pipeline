pipeline
========

A microservice that does micro things.

## Building

`make`


## Running

`./bin/pipeline`

## License
Copyright (c) 2014-2016 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Pipeline git struct
Rancher CI service will use git server(Gogs) to storage Pipeline File. Git repository structure will be like following:

```
-- templates
  |-- <Pipeline name>
  |   |-- 0
  |   |   |-- pipeline.yml
  |   |-- 1
  |   |   |-- pipeline.yml
...
```
pipeline.yml file describes the process of CI and CD. A pipeline includes several stages and steps. The first stage and first step will be the build stage and the build step.
User can have their test, deploy and deliver stages and steps after build step.

## Pipeline file example
```yaml
---
name: test1
repository: http://github.com/orangedeng/ui.git
branch: master
target-image: rancher/ui:v0.1
stages:
  - name: build
    need-approve: false
    steps:
    - name: build
      image: test/build:v0.1
      type: task
      command: make
      parameters:
      - "env=dev"
  - name: test
    need-approve: false
    steps:
    - name: source code check
      image: test/test:v0.1
      command: echo 'i am test'
      type: task
    - name: run server test
      image: test/run-bin:v0.1
      command: /startup.sh
      type: task
    - name: API test 
      image: test/api-test:v0.1
      command: /startup.sh && /api_test.sh
      type: task
  - name: deploy to test environment
    need-approve: true
    steps:
    - name: deploy a mysql
      type: catalog
      environment: 1a5
      docker-compose: |
        ...
        ...
      rancher-compose: |
        ...
        ...
    - name: deploy app
      type: deploy
      deploy-environment: 1a5
      deploy-name: app1
      count: 2
```