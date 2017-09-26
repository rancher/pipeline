from common import *  # NOQA

TEST_BUILD = os.environ.get('TEST_BUILD', 'false')

if_test_build = pytest.mark.skipif(
    TEST_BUILD != "true",
    reason='TEST_BUILD is set to false')


@if_test_build
def test_run_build():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "dockerfilePath": "",
                "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github",
                "type": "scm"}]
        },
        {
            "name": "buildfromsource",
            "steps": [
                {
                    "dockerfilePath": "./",
                    "isShell": False,
                    "sourceType": "sc",
                    "targetImage": "abcde:mytag",
                    "type": "build"}]
        },
        {
            "name": "buildfromfile",
            "steps": [{
                "dockerfilePath": "./",
                "file": "FROM alpine\n\nRUN echo test \u003e /myfile",
                "isShell": False,
                "sourceType": "file",
                "targetImage": "abcde:mytag",
                "type": "build"}]
        },
        {
            "name": "push",
            "steps": [{
                "dockerfilePath": "./",
                "isShell": False,
                "push": True,
                "sourceType": "sc",
                "targetImage": "reg.cnrancher.com/rancher/pipeline/" +
                               "test:nocool",
                "type": "build"}]
        }]
    test_registry_server = os.environ.get('TEST_REG_SERVER')
    test_registry_username = os.environ.get('TEST_REG_USERNAME')
    test_registry_password = os.environ.get('TEST_REG_PASSWORD')
    if test_registry_server is None or test_registry_username is None or \
       test_registry_password is None:
        print('No registry credential passed to test build')
        assert 0
    else:
        create_reg_cred(test_registry_server,
                        test_registry_username,
                        test_registry_password)
    pipeline = create_pipeline(name='buildtest', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('buildtest', 'Success')
    remove_pipeline('hello')
    remove_reg_cred(test_registry_server)
