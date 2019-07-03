# Seed Cluster Network Test

The 'seed_cluster_network_test.py' is a connectivity test for the seed 
cluster can identify network problems. We have seen those problems a 
couple of times:

- a node appears to be isolated in the cluster and cannot be reached
  from the other cluster nodes
- api servers might not be able to connect to their api server 


```
$ ./seed-cluster-network-test.py
usage: seed-cluster-network-test.py [-h] [--nodes] [--control-planes]
                                    [--seeds SEEDS]

Seed cluster connectivity test.

optional arguments:
  -h, --help        show this help message and exit
  --nodes           node connectivity test
  --control-planes  control plane components connectivity test
  --seeds SEEDS     seed cluster namespace (seed--<project>-->name>
```

# Known limitations

- The test will react sensitiv to changes in the cluster. It will
  most likely fail if control planes are removed, api servers are 
  being scaled down or nodes are being removed while the test runs.
- At the time of writing this test we did not see any problems 
  that this test is supposed to identifiy so the error handling and 
  reporting might not be optimal.

# ```network-ping-test.sh```

This script is kept for historical reasons. Error handling does not 
exist and it is incredibly slow for large clusters.