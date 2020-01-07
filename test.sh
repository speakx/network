#!/bin/bash

# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
# 测试选项
if [ ! -n "$1" ] ;then
    echo "you need input test target { all | bufpool }."
    exit
else
    echo "the test target is $1"
    echo
fi
target=$1
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # # # #

org=${PWD%/*}
org=${org##*/}
repository=${PWD##*/}
echo "** org:$org"
echo "** repository:$repository"
echo

# 重新造一遍 go mod
sh ./shell/gen-proto.sh
sh ./shell/configure.sh

if [ "$target" == "all" ] || [ "$target" == "bufpool" ] ;then
    cd ./src
    go test -v  ./bufpool/circleslidingbuffer_test.go ./bufpool/circleslidingbuffer.go ./bufpool/slidingbuffer.go ./bufpool/pool.go
fi
