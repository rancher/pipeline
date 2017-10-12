from common_fixtures import *  # NOQA
import gdapi
import cattle
import pytest

DEFAULT_TIMEOUT = 180
api_version = "v3"


@pytest.fixture(scope='session')
def rancher_client():
    # ACCESS_KEY = os.environ.get('ACCESS_KEY')
    # SECRET_KEY = os.environ.get('SECRET_KEY')
    # PROJECT_ID = os.environ.get('PROJECT_ID', "1a5")
    c = cattle.from_env(url=cattle_url(PROJECT_ID),
                        cache=False,
                        access_key=ACCESS_KEY,
                        secret_key=SECRET_KEY)
    return c


@pytest.fixture(scope='session')
def pipeline_client():
    PIPELINE_URL = os.environ.get('PIPELINE_URL', 'http://localhost:60080/v1')
    return gdapi.from_env(url=PIPELINE_URL,
                          access_key=ACCESS_KEY,
                          secret_key=SECRET_KEY)


def wait_for_ready_pipeline_client(timeout=DEFAULT_TIMEOUT):
    start = time.time()
    while True:
        try:
            pipeline_client()
        except:
            if time.time() - start > timeout:
                raise Exception('Fail connect pipeline server')
            else:
                time.sleep(10)
        else:
            return pipeline_client()


def cattle_url(project_id=None):
    default_url = 'http://localhost:8080'
    server_url = os.environ.get('CATTLE_TEST_URL', default_url)
    server_url = server_url + "/" + api_version
    if project_id is not None:
        server_url += "/projects/"+project_id
    return server_url


def ensure_cicd_catalog():
    if cicd_is_up() == 1:
        return
    else:
        t_env = {
            "SLAVES": 0,
            "EXECUTORS": 2,
            "JENKINS_PORT": 8081,
            "VOLUME_DRIVER": "local",
            "HOST_LABEL": ""
        }
        deploy_cicd(t_env)
        wait_for_ready_pipeline_client()


@pytest.fixture()
def pipeline_resource(request):
    ensure_cicd_catalog()

    def cleanup():
        cleanup_activity()
        cleanup_pipeline()
    request.addfinalizer(cleanup)


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


def create_pipeline(**kw):
    return pipeline_client().create_pipeline(**kw)


def cleanup_pipeline():
    pipelines = pipeline_client().list_pipeline()
    for p in pipelines:
        p.remove()


def get_activities():
    return pipeline_client().list_activity()


def get_activity(id):
    return pipeline_client().by_id_activity(id)


def update_setting(**kw):
    return pipeline_client().list_setting().update(**kw)


def cleanup_activity():
    activities = pipeline_client().list_activity()
    for a in activities:
        a.remove()


def get_setting():
    return pipeline_client().list_setting()


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


def run_basic_pipeline():
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "repository": "https://github.com/gitlawr/sh.git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('hello', 'Success')
    remove_pipeline('hello')


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
            time.sleep(3)
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


def deploy_and_wait_for_stack(client,
                              dockerCompose,
                              rancherCompose,
                              environment,
                              t_name,
                              externalId,
                              system):
    env = client.create_stack(name=t_name,
                              dockerCompose=dockerCompose,
                              rancherCompose=rancherCompose,
                              environment=environment,
                              startOnCreate=True,
                              externalId=externalId,
                              system=system)
    env = client.wait_success(env, timeout=300)
    wait_for_condition(
        client, env,
        lambda x: x.healthState == "healthy",
        lambda x: 'State is: ' + x.state,
        timeout=600)
    for service in env.services():
        wait_for_condition(
            client, service,
            lambda x: x.state == "active",
            lambda x: 'State is: ' + x.state,
            timeout=600)
        container_list = get_service_container_list(client, service,
                                                    managed=1)
        for container in container_list:
            if 'io.rancher.container.start_once' not in container.labels:
                assert container.state == "running"
            else:
                assert \
                    container.state == "stopped" or \
                    container.state == "running"


def deploy_cicd(t_env):
    auth = (ACCESS_KEY, SECRET_KEY)
    t_version = os.environ.get('CICD_CATALOG_VERSION', 7)
    t_name = "CICD"
    externalId = "catalog://CICD:infra*CICD:" + str(t_version)
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
    deploy_and_wait_for_stack(rancher_client(),
                              dockerCompose,
                              rancherCompose,
                              t_env,
                              t_name,
                              externalId,
                              system)


def remove_cicd():
    rcli = rancher_client()
    t_name = 'CICD'
    env = rcli.list_stack(name=t_name)
    for i in range(len(env)):
        delete_all(rcli, [env[i]])
    print 'removed cicd catalog'


def cicd_is_up():
    t_name = 'CICD'
    env = rancher_client().list_stack(name=t_name)
    return len(env) == 1


def get_hosts():
    hosts = rancher_client().list_host()
    for host in hosts:
        assert host["uuid"] is not None
        print(host["uuid"])
        print(host["hostname"])
    return


def create_registry(serverAddress, username, password):
    c = rancher_client()
    reg = c.create_registry(serverAddress=serverAddress)
    c.create_registryCredential(registryId=reg.id,
                                publicValue=username,
                                secretValue=password)


def remove_registry(serverAddress):
    regs = rancher_client().list_registry()
    for reg in regs:
        if reg.serverAddress == serverAddress:
            reg.remove()


def get_registry(serverAddress):
    regs = rancher_client().list_registry()
    for reg in regs:
        if reg.serverAddress == serverAddress:
            return reg
    return None


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
