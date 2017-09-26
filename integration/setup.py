from distutils.core import setup

setup(
    name='CICD Integration Tests',
    version='0.1',
    packages=[
      'cicd',
    ],
    license='ASL 2.0',
    long_description=open('README.txt').read(),
)