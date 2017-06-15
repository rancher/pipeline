FROM ubuntu:16.04
COPY pipeline /usr/bin/
RUN mkdir -p /data/rancher_ci/templates /data/rancher_ci/jenkins /data/rancher_ci/workspaces  
ADD templates /templates
ADD jenkins /jenkins
CMD ["pipeline"]
