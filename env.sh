#!/bin/bash

repository=${PWD##*/}
echo "依赖工程"

echo "singledb"
cd ../singledb
sh ./shell/gen-proto.sh
cd ../$repository

echo "single"
cd ../single
sh ./shell/gen-proto.sh
cd ../$repository
