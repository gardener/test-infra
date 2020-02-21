# Telemetry Controller

The telemetry controller is a kubernetes controller that wraps the shoot telemetry controller with a CRD.
With the telemetry controller the measurement of shoots can be instructed using a ShootsMeasurements CRD.

### Development
```
go run ./cmd/telemetry-controller/main.go
  --cache-dir=/tmp/tl # specify the directory path for the measurements cache 
  --kubeconfig=/.kube/config # optional, uses the in cluster config if not defined
```

### ShootsMeasurements

A measurement can be started by posting the following CRD into a cluster running the shoots measurement controller.
The measurement can be stopped by annotating the CRD with `"telemetry.testmachinery.gardener.cloud/stop"="true"`.
```
kubectl annotate sm <name> "telemetry.testmachinery.gardener.cloud/stop"="true"
```

```yaml
apiVersion: telemetry.testmachinery.gardener.cloud/v1beta1
kind: ShootsMeasurement
metadata:
  generateName: example
  namespace: default
spec:
  gardenerSecretRef: "garden-dev" # secret name in the same ns as the measurements file

  shoots: # list of shoots to measure
  - Name: ts-test
    Namespace: garden-core
status:
  data:
  - countRequest: 6180
    countRequestTimeouts: 2
    countUnhealthyPeriods: 0
    provider: gcp
    responseTimesMs:
      avg: 12
      max: 78
      median: 10
      min: 1
      std: 7.044667932437778
    seed: gcp
    shoot:
      Name: test
      Namespace: garden-core
  message: Successfully measures 1 shoot
  observedGeneration: 1
  phase: Completed
```
