#!/usr/bin/env bash

cd /usr/testbed/sbin

getent passwd adama
if [ 0 -ne "$?" ]; then 
  ./wap ./newuser /tmp/config/adama.xml
  ./wap ./newproj /tmp/config/galactica.xml
  ./wap ./mkproj galactica
  ./wap ./tbacct add adama
  mysql tbdb -e "update users set status='active' where uid='adama'"
  mysql tbdb -e "update sitevariables set value=NULL where name='general/firstinit/state'"
fi


#TODO add adama to boss?
#/usr/sbin/pw useradd \
#  $protouser -u $uid -g $agid \
#	   -G $Ggid -h - \
#	   -m -d $HOMEDIR/$protouser -s $binshell \
#	   -c \"$protouser_name
