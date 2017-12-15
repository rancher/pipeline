FROM ubuntu:16.04

MAINTAINER Lawrence <lawrence@rancher.com>

RUN apt-get update && apt-get -y upgrade && apt-get -y install software-properties-common && add-apt-repository ppa:webupd8team/java -y && apt-get update

RUN (echo oracle-java8-installer shared/accepted-oracle-license-v1-1 select true | /usr/bin/debconf-set-selections) && apt-get install -y oracle-java8-installer oracle-java8-set-default

ENV JAVA_HOME /usr/lib/jvm/java-8-oracle
ENV PATH $JAVA_HOME/bin:$PATH


# apparmor is required to run docker server within docker container
RUN apt-get update -qq && apt-get install -qqy wget curl git iptables ca-certificates apparmor

ENV JENKINS_SWARM_VERSION 2.2
ENV HOME /home/jenkins-slave


RUN useradd -c "Jenkins Slave user" -d $HOME -m jenkins-slave
RUN curl --create-dirs -sSLo $HOME/swarm-client-$JENKINS_SWARM_VERSION-jar-with-dependencies.jar https://repo.jenkins-ci.org/releases/org/jenkins-ci/plugins/swarm-client/$JENKINS_SWARM_VERSION/swarm-client-$JENKINS_SWARM_VERSION-jar-with-dependencies.jar
ADD cmd.sh /cmd.sh

# set our wrapper
ENTRYPOINT ["/usr/local/bin/docker-wrapper"]

# setup our local files first
ADD docker-wrapper.sh /usr/local/bin/docker-wrapper
RUN chmod +x /usr/local/bin/docker-wrapper

# now we install docker in docker - thanks to https://github.com/jpetazzo/dind
# We install newest docker into our docker in docker container
RUN curl -fsSLO https://get.docker.com/builds/Linux/x86_64/docker-latest.tgz \
  && tar --strip-components=1 -xvzf docker-latest.tgz -C /usr/local/bin \
  && chmod +x /usr/local/bin/docker \
# install Rancher CLI
  && curl -fsSLO https://github.com/rancher/cli/releases/download/v0.4.1/rancher-linux-amd64-v0.4.1.tar.gz \
  && tar --strip-components=2 -xzvf rancher-linux-amd64-v0.4.1.tar.gz -C /usr/local/bin \
  && chmod +x /usr/local/bin/rancher

RUN git config --global user.email "ci@rancher.com" &&git config --global user.name "rancher_ci"

# add tools for cicd
# https://github.com/rancher/cli
ADD rancher /usr/local/bin/rancher
# https://github.com/gitlawr/mergeyaml
ADD mergeyaml /usr/local/bin/mergeyaml
# https://github.com/gitlawr/rancher-upgrader
ADD rancher-upgrader /usr/local/bin/rancher-upgrader

VOLUME /var/lib/docker

ENV JENKINS_USERNAME admin
ENV JENKINS_PASSWORD admin
ENV JENKINS_MASTER http://jenkins:8080

CMD /bin/bash /cmd.sh
