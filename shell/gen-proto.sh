#!/bin/bash

org=${PWD%/*}
org=${org##*/}
repository=${PWD##*/}

function rename_proto_package() {
    newname=$1".tmp"
    sed "s/ *package *[a-zA-Z]*;/package pb$repository;/1" $1 >> $newname
    rm -rf ./$1
    mv $newname $1
}

function gen_proto() {
	protofolder=`ls | grep proto`
    if [ "$protofolder" != "" ] && [ -d "$(pwd)/$protofolder" ] ;then
        cd ./$protofolder

        folders=`ls`
		for folder in $folders; do
            if [ -d "$(pwd)/$folder" ]; then
                rm -rf "$(pwd)/$folder"
            fi
		done

        rm -rf ./*.proto.tmp
        protos=`ls | grep ".proto"`

		for proto in $protos; do
            rename_proto_package $proto
		done

        if [ "$protos" != "" ] ;then
            rm -rf ./*.pb.go
            rm -rf ./"pb"$repository
            protoc --go_out=plugins=grpc:. *.proto
            mkdir "pb"$repository
            mv *.go ./"pb"$repository
        fi 
        cd ../
    fi
	
    folders=`ls`
	for folder in $folders; do
		if [ -d "$(pwd)/$folder" ] ;then
			cd $folder
				gen_proto
			cd ../
		fi
	done
}

## build proto
echo "build proto..."
export PATH=$PATH:$GOPATH/bin
echo "PATH=$PATH"
echo
gen_proto
echo "gen proto done"
echo