#!/usr/bin/env bash

cd /usr/testbed/sbin

getent passwd adama
if [ 0 -ne "$?" ]; then 
  set -e
  ./wap ./newuser /tmp/config/adama.xml
  ./wap ./newproj /tmp/config/galactica.xml
  ./wap ./mkproj galactica
  ./wap ./tbacct add adama
  mysql tbdb -e "update sitevariables set value=NULL where name='general/firstinit/state'"
fi

