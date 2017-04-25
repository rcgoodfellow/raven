#!/bin/sh

cd /usr/testbed/sbin
./wap ./newuser /tmp/config/adama.xml
./wap ./newproj /tmp/config/galactica.xml
./wap ./mkproj galactica
mysql tbdb -e "update sitevariables set value=NULL where name='general/firstinit/state'"
