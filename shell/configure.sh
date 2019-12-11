#!/bin/bash

org=${PWD%/*}
org=${org##*/}
repository=${PWD##*/}
echo "** org:$org"
echo "** repository:$repository"
echo

go_mod_replace_sp="!"
go_mod_counter=0
go_mod_replace=()
function init_pkg_mod() {
	gomod=`ls | grep 'go.mod'`
	if [ "$gomod" != "" ] ;then
		mod=${PWD##*/pkg/}
		if [ "$mod" != "$org/$repository" ] ;then
			go_mod_replace[$go_mod_counter]="replace$go_mod_replace_sp$mod$go_mod_replace_sp=>$go_mod_replace_sp../../pkg/$mod"
			go_mod_counter=$(($go_mod_counter+1))
		fi
	else
		folders=`ls`
		for folder in $folders; do
			if [ -d "$(pwd)/$folder" ] ;then
				cd $folder
					init_pkg_mod
				cd ../
			fi
		done
	fi
}

function replace_go_mod() {
	gomod=`ls | grep 'go.mod'`
	if [ "$gomod" != "" ] ;then
		modname=`cat go.mod | grep module | awk '{print $2}'`
		if [ "$modname" != "$repository" ] ;then
			echo $(pwd)"/go.mod vs $1"
			mod=`cat go.mod | grep module | awk '{print $2}'`
			curdir=$(pwd)
			target_mod_dir=${curdir//$repositorydir/..\/..}
			go_mod_replace[$go_mod_counter]="replace$go_mod_replace_sp$mod$go_mod_replace_sp=>$go_mod_replace_sp$target_mod_dir"
			go_mod_counter=$(($go_mod_counter+1))
		fi
	else
		folders=`ls`
		for folder in $folders; do
			if [ -d "$(pwd)/$folder" ] && [ $folder != "pkg" ] ;then
				cd $folder
				replace_go_mod $1
				cd ../
			fi
		done
	fi
}

echo "configure start..."
cd ./src
moddir=$(pwd)
rm -f go.mod
rm -f go.sum
echo "go mod init $repository"
go mod init $repository
cd ../
echo

cd ../
repositorydir=$(pwd)
replace_go_mod $moddir
cd ./$repository
echo "pwd : "$(pwd)

# check if ./pkg is exist.
echo "go pkg check..."
pkgfolder=`ls ../ | grep pkg`
if [ "$pkgfolder" != "" ] ;then
	cd ../pkg
	init_pkg_mod
	cd ../$repository
fi


echo "go mod replace"
cd ./src
for replace in ${go_mod_replace[@]};do
	r=${replace//$go_mod_replace_sp/ }
	echo $r" >> go.mod"
	echo $r >> go.mod
done
cd ../

echo "go mod tidy"
cd ./src
go mod tidy
cd ../

echo "configure done..."
echo
