#!/bin/bash

repository=${PWD##*/}
targetos=`uname | tr "[A-Z]" "[a-z]"`
sh ./build.sh $targetos
echo

rm -f ./log/*
echo "./bin/$repository $1 $2 $3 $4"
./bin/$repository $1 $2 $3 $4
