from common_fixtures import *  # NOQA
import gdapi
import cattle
pytest
CATTLE_URL = 'http://47.52.106.231:8080/v1/schemas'
CATALOG_URL = 'http://localhost:8082/v1-catalog'
PIPELINE_URL = 'http://localhost:60080/v1'

DEFAULT_TIMEOUT = 180


def rancher_client():
    # ACCESS_KEY = os.environ.get('ACCESS_KEY')
    # SECRET_KEY = os.environ.get('SECRET_KEY')
    # PROJECT_ID = os.environ.get('PROJECT_ID', "1a5")
    c = cattle.from_env(url=cattle_url(PROJECT_ID),
                        cache=False,
                        access_key=ACCESS_KEY,
                        secret_key=SECRET_KEY)
    return c


def pipeline_client():
    PIPELINE_URL = os.environ.get('PIPELINE_URL', 'http://localhost:60080/v1')
    return gdapi.from_env(url=PIPELINE_URL)


def get_pipelines():
    pipelines = pipeline_client().list_pipeline()
    for p in pipelines:
        assert p['name'] is not None
        # print('pipeline_id: ', p['id'])
        # print('pipeline_name: ', p['name'])
        # print('pipeline_stages: ', p['stages'])
    return pipelines


def get_pipeline(id):
    return pipeline_client().by_id_pipeline(id)


def get_activities():
    return pipeline_client().list_activity()


def get_activity(id):
    return pipeline_client().by_id_activity(id)


def create_pipeline(**kw):
    return pipeline_client().create_pipeline(**kw)


def remove_pipeline(name):
    pipelines = pipeline_client().list_pipeline()
    found = False
    for p in pipelines:
        assert p['name'] is not None
        if p['name'] == name:
            p.remove()
            found = True
    assert found, 'Fail remove pipeline, not found'


def run_pipeline(name):
    pipelines = pipeline_client().list_pipeline()
    found = False
    for p in pipelines:
        assert p['name'] is not None
        if p['name'] == name:
            p.run()
            found = True
    assert found, 'Fail run pipeline, not found'


def run_pipeline_expect(name, status, timeout=DEFAULT_TIMEOUT):
    pipelines = pipeline_client().list_pipeline()
    found = False
    for p in pipelines:
        assert p['name'] is not None
        if p['name'] == name:
            a = p.run()
            found = True
            expect_activity_status(a.id, status, timeout)
    assert found, 'Fail run pipeline, not found'


def wait_activity_expect(id, status, timeout=DEFAULT_TIMEOUT):
    expect_activity_status(id, status, timeout)


def expect_activity_status(activityid, status,
                           timeout=DEFAULT_TIMEOUT, timeout_message=None):
    start = time.time()
    while True:
        activity = pipeline_client().by_id_activity(activityid)
        assert activity.status is not None
        if activity.status == 'Building' or activity.status == 'Running' \
           or activity.status == 'Waiting':
            time.sleep(.5)
        elif activity.status == status:
            return
        else:
            raise Exception('Expected status ' + status + ' for activity ' +
                            activity.id + ' but got ' + activity.status)
        if time.time() - start > timeout:
            if timeout_message:
                raise Exception(timeout_message)
            else:
                raise Exception('Timeout waiting for activity ' +
                                activity.id + ' ' + activity.status)


def deploy_cicd():
    auth = (ACCESS_KEY, SECRET_KEY)
    t_name = "CICD"
    t_version = 7
    t_env = {
        "SLAVES": 0,
        "EXECUTORS": 2,
        "JENKINS_PORT": 8081,
        "VOLUME_DRIVER": "local",
        "HOST_LABEL": ""
        }
    system = True
    headers = {}
    headers["X-API-Project-Id"] = PROJECT_ID
    catalog_url = rancher_server_url() + "/v1-catalog/templates/CICD:infra*"
    # Deploy Catalog template from catalog
    print(catalog_url + t_name + ":" + str(t_version))
    print(auth)
    print(headers)
    r = requests.get(catalog_url + t_name + ":" + str(t_version),
                     auth=auth,
                     headers=headers)
    template = json.loads(r.content)
    r.close()
    dockerCompose = template["files"]["docker-compose.yml"]
    rancherCompose = template["files"]["rancher-compose.yml"]
    print(t_env)
    print(t_name)
    print(system)
    deploy_and_wait_for_stack_creation(rancher_client(),
                                       dockerCompose,
                                       rancherCompose,
                                       t_env,
                                       t_name,
                                       system)
    print 'deployed cicd catalog'


def remove_cicd():
    t_name = 'CICD'
    env = rancher_client().list_stack(name=t_name)
    for i in range(len(env)):
        delete_all(client, [env[i]])
    print 'removed cicd catalog'


def cicd_is_up():
    t_name = 'CICD'
    env = rancher_client().list_stack(name=t_name)
    assert len(env) == 1


def get_hosts():
    hosts = rancher_client().list_host()
    for host in hosts:
        assert host["uuid"] is not None
        print(host["uuid"])
        print(host["hostname"])
    return


def create_reg_cred(serverAddress, username, password):
    c = rancher_client()
    reg = c.create_registry(serverAddress=serverAddress)
    c.create_registryCredential(registryId=reg.id,
                                publicValue=username,
                                secretValue=password)


def remove_reg_cred(serverAddress):
    c = rancher_client()
    regs = c.list_registry()
    for reg in regs:
        if reg.serverAddress == serverAddress:
            reg.remove()


def get_catalog_templates():
    templates = rancher_client().list_template(projectId='1a5',
                                               catalogId='CICD')
    for template in templates:
        print(template["id"])
        print(template["description"])
    return


def main():
    print 'main start'


if __name__ == "__main__":
    main()
