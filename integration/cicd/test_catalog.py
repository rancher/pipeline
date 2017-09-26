from common import deploy_cicd, remove_cicd


def no_test_catalog():
    deploy_cicd()
    remove_cicd()
