#!/bin/bash

# ping all nodes from all calico pods (one per node) and print output
# of ping command in case the ping operation fails

hosts=$(kubectl get nodes -o jsonpath="{.items[*].status.addresses[0].address}")
calico_pods=$(kubectl get pods -n kube-system | grep calico-node | awk '{ print $1 }')

for cp in $calico_pods; do 

    for host in $hosts; do
        echo "Pinging node ${host} from calico pod ${cp}"
        out=$(kubectl -n kube-system exec -it ${cp} -c calico-node -- ping -c 4 $host)
        if [ $? -ne 0 ] ; then
            echo "Problem pinging node ${host} from calico pod ${cp}"
            echo $out
        else
            echo "ok"
        fi
        echo
    done
done
