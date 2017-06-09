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

## Pipeline file example
```yaml
---
name: test1
repository: http://github.com/orangedeng/ui.git
branch: master
target_image: rancher/ui:v0.1
stages:
  - name: stage zero
    need_approve: false
    steps:
    - name: step zero
      image: test/build:v0.1
      command: make
      parameters:
      - "env=dev"
  - name: stage test
    need_approve: false
    steps:
    - name: source code check
      image: test/test:v0.1
      command: echo 'i am test'
    - name: server run test
      image: test/run-bin:v0.1
      command: /startup.sh
    - name: API test 
      image: test/api-test:v0.1
      command: /startup.sh && /api_test.sh
```