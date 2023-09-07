#!/bin/sh

root_path=$(pwd)
sub_path=../broker
for folder in `find $sub_path/* -type d`
do
    cur_path=`realpath $root_path/$folder`
    echo $cur_path
    cd $cur_path
    go get all
    go mod tidy
done
