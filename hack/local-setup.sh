#!/usr/bin/env bash
set -e

NAMESPACE="default"

for i in "$@"
do
case $i in
    -h|--help)
    dump_help
    shift
    ;;
    -n=*|--namespace=*)
    NAMESPACE="${i#*=}"
    exit 0
    shift
    ;;
    *)
        # unknown option
        echo "Unknown option ${i#*=}"
        dump_help
        exit 1
    ;;
esac
done

function dump_help() {
  echo -e "Example usage: $(basename $0) -n garden-it -o 48"
  echo -e "   [-h | --help]. . . . . . . . . . . . . . . . . . . . . . Show this help."
  echo -e "   [-n | --namespace]. . . . . . . . . . . Namespcae where the testmachinery should be installed"
}


NS=$NAMESPACE make install-prerequisites

make run-controller