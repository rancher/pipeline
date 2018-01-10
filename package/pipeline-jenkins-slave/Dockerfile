FROM rancher/jenkins-slave

ARG SLAVE_VERSION

# setup our local files first
ADD docker-wrapper.sh /usr/local/bin/docker-wrapper
ADD wait-for-master.sh /usr/local/bin/wait-for-master
RUN chmod +x /usr/local/bin/docker-wrapper && \
    chmod +x /usr/local/bin/wait-for-master

# add tools for cicd
ADD https://github.com/rancher/cihelper/releases/download/v${SLAVE_VERSION}/cihelper /usr/local/bin/cihelper
RUN chmod +x /usr/local/bin/cihelper

CMD wait-for-master /bin/bash /cmd.sh
