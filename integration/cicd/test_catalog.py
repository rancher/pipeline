from common import *  # NOQA

TEST_CATALOG = os.environ.get('TEST_CATALOG', 'false')

if_test_catalog = pytest.mark.skipif(
    TEST_CATALOG != "true",
    reason='TEST_CATALOG is set to false')


@pytest.fixture(autouse=True)
def ensure_no_cicd_catalog():
    if cicd_is_up() == 1:
        remove_cicd()


@if_test_catalog
def test_default_cicd_catalog():
    t_env = {
        "SLAVES": 0,
        "EXECUTORS": 2,
        "JENKINS_PORT": 8081,
        "VOLUME_DRIVER": "local",
        "HOST_LABEL": ""
        }
    deploy_cicd(t_env)
    wait_for_ready_pipeline_client()
    remove_cicd()


@if_test_catalog
def test_slave_cicd_catalog():
    t_env = {
        "SLAVES": 1,
        "EXECUTORS": 2,
        "JENKINS_PORT": 8081,
        "VOLUME_DRIVER": "local",
        "HOST_LABEL": ""
        }
    deploy_cicd(t_env)
    wait_for_ready_pipeline_client()
    remove_cicd()
