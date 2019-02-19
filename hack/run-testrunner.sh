# /bin/bash
set -e

tm_kubeconfig_path=
testruns_chart_path=
testrun_prefix=test-

gardener_kubeconfig_path=
component_descriptor_path=""
es_config_name=""

go run cmd/testmachinery-run/main.go \
    --tm-kubeconfig-path=$tm_kubeconfig_path \
    --testruns-chart-path=$testruns_chart_path \
    --testrun-prefix=$testrun_prefix \
    --gardener-kubeconfig-path=$gardener_kubeconfig_path \
    --component-descriptor-path=$component_descriptor_path \
    --es-config-name=$es_config_name