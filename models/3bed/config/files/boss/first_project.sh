#!/usr/bin/env bash

cd /usr/testbed/sbin

getent passwd adama
if [ 0 -ne $? ]; then 
  ./wap ./newuser /tmp/config/adama.xml
fi

getent group galactica
if [ 0 -ne $? ]; then 
  ./wap ./mkproj galactica
  ./wap ./newproj /tmp/config/galactica.xml
fi

mysql tbdb -e "update sitevariables set value=NULL where name='general/firstinit/state'"
