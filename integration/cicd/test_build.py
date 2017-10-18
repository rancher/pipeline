from common import *  # NOQA

TEST_BUILD = os.environ.get('TEST_BUILD', 'false')

if_test_build = pytest.mark.skipif(
    TEST_BUILD != "true",
    reason='TEST_BUILD is set to false')


@if_test_build
def test_run_build(pipeline_resource):
    test_registry_server = os.environ.get('TEST_REG_SERVER')
    test_registry_username = os.environ.get('TEST_REG_USERNAME')
    test_registry_password = os.environ.get('TEST_REG_PASSWORD')
    test_build_image = os.environ.get('TEST_BUILD_IMAGE')
    if test_registry_server is None or test_registry_username is None or \
       test_registry_password is None or test_build_image is None:
        print 'No registry credential passed to test build'
        assert 0
    else:
        if get_registry(test_registry_server) is None:
            create_registry(test_registry_server,
                            test_registry_username,
                            test_registry_password)
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "repository": "https://github.com/gitlawr/php.git",
                "type": "scm"}]
        },
        {
            "name": "buildfromsource",
            "steps": [
                {
                    "dockerfilePath": "./",
                    "targetImage": test_build_image,
                    "type": "build"}]
        },
        {
            "name": "buildfromfile",
            "steps": [{
                "dockerfilePath": "./",
                "file": "FROM alpine\n\nRUN echo test \u003e /myfile",
                "targetImage": test_build_image,
                "type": "build"}]
        },
        {
            "name": "push",
            "steps": [{
                "dockerfilePath": "./",
                "push": True,
                "targetImage": test_build_image,
                "type": "build"}]
        }]
    create_pipeline(name='buildtest', stages=stages)
    run_pipeline_expect('buildtest', 'Success', 300)
    remove_pipeline('buildtest')


@if_test_build
def test_run_pipeline_build_fail_no_credential(pipeline_resource):
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master",
                "repository": "https://github.com/gitlawr/php.git",
                "type": "scm"}]
        },
        {
            "name": "buildfromsource",
            "steps": [
                {
                    "dockerfilePath": "./",
                    "push": True,
                    "targetImage": "nonexist.com/myimage:mytag",
                    "type": "build"}]
        }]
    create_pipeline(name='buildfailpushtest', stages=stages)
    run_pipeline_expect('buildfailpushtest', 'Fail')
    remove_pipeline('buildfailpushtest')
