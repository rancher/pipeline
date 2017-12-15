FROM busybox
ENV BOOT_VERSION 1.0.0
RUN wget -q -O /boot.tar.gz https://www.github.com/biblesyme/jenkins-boot/archive/v${BOOT_VERSION}.tar.gz
RUN mkdir /var/rancher \
    && tar -xzf /boot.tar.gz \
    && mv /jenkins-boot-${BOOT_VERSION}/jenkins_home /var/rancher
COPY ./cpjenkins.sh /
RUN mkdir /var/jenkins_home
VOLUME /var/jenkins_home
CMD ["sh", "./cpjenkins.sh"]
