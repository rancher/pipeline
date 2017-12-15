#!/bin/bash
# If mount an empty dir, then copy the initial home

DIR="/var/jenkins_home"
if [ "$(ls -A $DIR)" ];
then
     echo "$DIR is not Empty"
else
     echo "$DIR is Empty"
     echo "Start copy jenkins_home by running cp -r /var/rancher/jenkins_home/* $DIR"
     cp -r /var/rancher/jenkins_home/* $DIR
     echo "Copy Finishied"
fi
