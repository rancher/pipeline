FROM busybox
ARG BOOT_VERSION
RUN wget -q -O /boot.tar.gz https://www.github.com/rancher/jenkins-boot/archive/v${BOOT_VERSION}.tar.gz
RUN mkdir /var/rancher \
    && tar -xzf /boot.tar.gz \
    && mv /jenkins-boot-${BOOT_VERSION}/jenkins_home /var/rancher
COPY ./cpjenkins.sh /
RUN mkdir /var/jenkins_home
VOLUME /var/jenkins_home
CMD ["sh", "./cpjenkins.sh"]
