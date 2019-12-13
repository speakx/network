#!/bin/bash

repository=${PWD##*/}
targetos=`uname | tr "[A-Z]" "[a-z]"`
sh ./build.sh $targetos
echo

if [ "$1" != "-client" ] ;then
    rm -f ./log/network.log.server*
else
    rm -f ./log/network.log.client*
fi
echo "./bin/$repository $1 $2 $3 $4"
./bin/$repository $1 $2 $3 $4
