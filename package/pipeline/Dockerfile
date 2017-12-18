FROM ubuntu:16.04
ADD zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
RUN apt-get update && apt-get install -y curl ca-certificates git && rm -rf /var/lib/apt/lists/*
COPY pipeline /usr/bin/
CMD ["pipeline"]
