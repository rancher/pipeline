FROM ubuntu:16.04
COPY pipeline /usr/bin/
RUN mkdir -p /data/rancher-ci/templates /data/rancher-ci/jenkins /data/rancher-ci/workspace
ENV JENKINS_HOME /data/rancher-ci/workspace
CMD ["pipeline"]
