#!/bin/sh

root_path=$(pwd)
sub_path=../_example

update_go_mod() {
  cd $root_path
  for folder in `find $sub_path/$1/* -type d`
  do
      cur_path=`realpath $root_path/$folder`
      echo $cur_path
      cd $cur_path
      go get all
      go mod tidy
  done
}

update_go_mod "broker"
update_go_mod "server"
