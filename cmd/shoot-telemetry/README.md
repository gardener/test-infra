# Garden Shoot Telemetry Controller

A telemetry controller to get granular insights of Shoot apiserver and etcd availability.

The measurements will be persistent by appending a `results.csv` file in the passed output directory.
The controller is keeping the measurements for 30 seconds in memory, before it appends the data to the `result.csv`.

**Disclaimer: Please keep in mind this is still on a prototype level**

### Build and Run
```sh
# Build
make build

# Run
./bin/garden-shoot-telemetry-<linux|darwin>-amd64 \
  --kubeconfig <path-to-kubeconfig-for-garden-cluster> \
  --output <directory-to-write-measurements-csv-file> \
  --interval 5s
```

### Analyse the Data
When the controller process receives a SIGTERM signal it writes the remaining data in memory to disk.
After that the analyse functionality will be invoked, which will calculate and print statistical key figures like min/max, avg, etc. for the unhealthy periods of each cluster to stdout or into a passed file.

The analysis functionality can also be used to anlayse existing mesaurment files.
Lets check out the example data in `example/measurments.csv`.

The  analysis of the example data can be manually triggered by running the following command:
```sh
./bin/garden-shoot-telemetry-<linux|darwin>-amd64 \
  analyse
  --input example/measurements.csv
```

The `example/measurements.csv` file contains data for one cluster with four unhealthy periods.

You should see the name of the cluster, the count of unhealthy periods, the shortest(min)/largest(max) unhealthy period and the average, median and standard deviation of the durations for the unhealthy periods of the clusters.