from common import *  # NOQA


def test_list_pipeline():
    get_pipelines()


def test_list_activity():
    get_activities()


def test_get_pipeline():
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    pipeline = get_pipeline(pipeline.id)
    assert pipeline is not None, 'Failed create pipeline.'
    remove_pipeline('hello')


def test_create_pipeline_basic():
    # cicd_is_up()
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    remove_pipeline('hello')


def test_activate_pipeline():
    # cicd_is_up()
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    pipeline.activate()
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.isActivate, "Expected IsActivate"
    pipeline.deactivate()
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.isActivate is False, "Expected IsActivate"
    remove_pipeline('hello')


def test_create_pipeline_fail_no_scm():
    with pytest.raises(Exception):
        stages = [{
            "name": "SCM",
            "steps": [
                    {
                        "branch": "master",
                        "dockerfilePath": "",
                        "isShell": False,
                        "repository": "https://github.com/gitlawr/sh.git",
                        "sourceType": "git",
                        "type": "task"
                    }
                ]
        }]
        pipeline = create_pipeline(name='hello', stages=stages)
        assert pipeline.id is not None, 'Failed create pipeline.'
        remove_pipeline('hello')


def test_create_pipeline_php():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "service",
            "steps": [{
                "alias": "mysql", "dockerfilePath": "",
                "image": "mysql:5.6", "isService": True,
                "isShell": False, "parameters": [
                    "MYSQL_DATABASE=hello_world_test",
                    "MYSQL_ROOT_PASSWORD=root"],
                "type": "task"}]
        },
        {
            "name": "test",
            "steps": [{
                "command": ("# Install git, the php image doe"
                            "sn't have installed\napt-get upd"
                            "ate -yqq\napt-get install git -y"
                            "qq\n\n# Install mysql driver\ndo"
                            "cker-php-ext-install pdo_mysql\n\n"
                            "# Install composer\ncurl -sS https"
                            "://getcomposer.org/installer | php"
                            "\n\n# Install all project dependen"
                            "cies\nphp composer.phar install\ve"
                            "ndor/bin/phpunit --configuration ph"
                            "punit_mysql.xml --coverage-text"),
                "dockerfilePath": "", "image": "php:5.6",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='phptest', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    remove_pipeline('phptest')


def test_run_basic():
    stages = [{
        "name": "SCM",
        "steps": [
                {
                    "branch": "master",
                    "dockerfilePath": "",
                    "isShell": False,
                    "repository": "https://github.com/gitlawr/sh.git",
                    "sourceType": "git",
                    "type": "scm"
                }
            ]
    }]
    pipeline = create_pipeline(name='hello', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('hello', 'Success')
    remove_pipeline('hello')


def test_run_php():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/php.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "service",
            "steps": [{
                "alias": "mysql", "dockerfilePath": "",
                "image": "mysql:5.6", "isService": True,
                "isShell": False, "parameters": [
                    "MYSQL_DATABASE=hello_world_test",
                    "MYSQL_ROOT_PASSWORD=root"],
                "type": "task"}]
        },
        {
            "name": "test",
            "steps": [{
                "command": ("# Install git, the php image doesn't"
                            " have installed\napt-get update -yqq"
                            "\napt-get install git -yqq\n\n# Ins"
                            "tall mysql driver\ndocker-php-ext-in"
                            "stall pdo_mysql\n\n# Install compose"
                            "r\ncurl -sS https://getcomposer.org/i"
                            "nstaller | php\n\n# Install all projec"
                            "t dependencies\nphp composer.phar inst"
                            "all\nvendor/bin/phpunit --configuration"
                            " phpunit_mysql.xml --coverage-text"),
                "dockerfilePath": "", "image": "php:5.6",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='phptest', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('phptest', 'Success', 600)
    remove_pipeline('phptest')


def test_run_fail_script():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "fail",
            "steps": [{
                "command": "echo failintest && false",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='tofail', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('tofail', 'Fail')
    remove_pipeline('tofail')


def test_run_pending():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='topending', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('topending', 'Pending')
    remove_pipeline('topending')


def test_approve_pending():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='toapprove', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('toapprove', 'Pending')
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.lastRunId is not None
    pipeline.approve()
    wait_activity_expect(pipeline.lastRunId, 'Success')
    remove_pipeline('toapprove')


def test_deny_pending():
    stages = [
        {
            "name": "SCM",
            "steps": [{
                "branch": "master", "dockerfilePath": "", "isShell": False,
                "repository": "https://github.com/gitlawr/sh.git",
                "sourceType": "github", "type": "scm"
                }]
        },
        {
            "name": "pend",
            "needApprove": True,
            "steps": [{
                "command": "echo pendintest",
                "dockerfilePath": "", "image": "busybox",
                "isShell": True, "type": "task"}]
        }]
    pipeline = create_pipeline(name='todeny', stages=stages)
    assert pipeline.id is not None, 'Failed create pipeline.'
    run_pipeline_expect('todeny', 'Pending')
    pipeline = get_pipeline(pipeline.id)
    assert pipeline.lastRunId is not None
    pipeline.deny()
    wait_activity_expect(pipeline.lastRunId, 'Denied')
    remove_pipeline('todeny')
