# Feature List

Pipeline:

| FEATURE                                  | DESC |
| ---------------------------------------- | ---- |
| get pipeline list                        |      |
| create pipeline                          |      |
| edit pipeline                            |      |
| clone pipeline                           |      |
| delete pipeline                          |      |
| activate/deactivate pipeline             |      |
| import/export pipeline                   |      |
| run pipeline                             |      |
| user-define parameter                    |      |
| pre-define environment variables         |      |
| environment variable sustitution in pipeline configuration |      |
| dragable stage/step on UI                |      |
| setting cron trigger for pipeline        |      |

Activity:

| FEATURE                                  | DESC |
| ---------------------------------------- | ---- |
| get activity list                        |      |
| rerun activity                           |      |
| stop activity                            |      |
| deny/approve `pending` activity          |      |
| get detail status of each stage and step |      |
| get log of a step                        |      |

Authorization:

| FEATURE                                  | DESC |
| ---------------------------------------- | ---- |
| Github Oauth for authorization           |      |
| Add/delete auth user                     |      |
| disable/reconfig github oauth application |      |
| share/unshare git account(Private by default) |      |

Stage:

| FEATURE                  | DESC |
| ------------------------ | ---- |
| set approval&approvers   |      |
| run in parallel/sequence |      |
| set conditions           |      |

Step:

| FEATURE                                  | DESC |
| ---------------------------------------- | ---- |
| scm - choose a git repo and branch       |      |
| scm - enable/disable github webhook      |      |
| task - run shell script                  |      |
| task - run custom entrypoint and commands |      |
| task - run as a service and be referenced by following steps |      |
| task - can pass  Environment Variables   |      |
| task - can use passed Environment Variables in shell script |      |
| build - build with Dockerfile in source code |      |
| build - build with uploaded Dockerfile   |      |
| build - push built image                 |      |
| upgradeService - upgrade image of selected services using service selector |      |
| upgradeService - upgrade preference by batch size, batch internal and replacement behavior |      |
| upgradeService - target other environment |      |
| upgradeStack - upgrade a stack with compose file |      |
| upgradeStack - target other environment  |      |
| upgradeCatalog - upgrade a catalog with template files |      |
| upgradeCatalog - upgrade a stack to newly updated catalog item version |      |
| upgradeCatalog - target other environment when upgrade a stack |      |