FROM nginx:1.12
ARG UI_VERSION
ADD https://www.github.com/rancher/pipeline-ui/releases/download/v${UI_VERSION}/${UI_VERSION}.tar.gz /dist.tar.gz
RUN tar -xzf /dist.tar.gz && mv /${UI_VERSION}/* /usr/share/nginx/html/
EXPOSE 80
