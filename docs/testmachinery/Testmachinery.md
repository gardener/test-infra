# Testmachinery


## Controller Arguments

| Name                  | Environment Variable | default                                                                 | Description                                                                                                                                     |
| --------------------- | -------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| namespace             | TM_NAMESPACE         | "default"                                                               | Namespace where the testmachinery-controller runs                                                                                               |
| prepare-image         | PREPARE_IMAGE        | "europe-docker.pkg.dev/gardener-project/releases/testmachinery/prepare-step:0.69.0" | Image that is used in the prepare step.                                                                                                         |
| base-image            | BASE_IMAGE           | "europe-docker.pkg.dev/gardener-project/releases/testmachinery/base-step:latest"    | Default image for test defintion with no explicit image.                                                                                        |
| enable-pod-gc         | CLEAN_WORKFLOW_PODS  | "false"                                                                 | Deletes all pods when a workflow is finished. Logs are still accessible.                                                                        |
| s3-endpoint           | S3_ENDPOINT          | ""                                                                      | Endpoint url of the minio/s3 instance where argo stores it's artifacts (e.g. minio-service.deafult:9000).                                       |
| s3-access-key         | S3_ACCESS_KEY        | ""                                                                      | Minio/S3 access key                                                                                                                             |
| s3-secret-key         | S3_SECRET_KEY        | ""                                                                      | Minio/S3 secret key                                                                                                                             |
| s3-bucket             | S3_BUCKET_NAME       | ""                                                                      | Minio/S3 bucket name, where argo stores it's artifacts.                                                                                         |
| github-secrets-path   | GIT_SECRETS          | ""                                                                      | Github configuration with technical users and endpoint information.                                                                             |
| webhook-http-address  | -                    | ":80"                                                                   | Webhook HTTP address to bind                                                                                                                    |
| webhook-https-address | -                    | ":443"                                                                  | Webhook HTTPS address to bind                                                                                                                   |
| cert-file             | -                    | ""                                                                      | Path to the server certificate                                                                                                                  |
| key-file              | -                    | ""                                                                      | Path to the private key corresponding to the server certificate.                                                                                |
| v                     | -                    | 0                                                                       | Specify the verbosity level of the logs.                                                                                                        |
| testdef-path          | TESTDEF_PATH         | ".test-defs"                                                            | Set repository path where the Test Machinery should search for testdefinition                                                                   |
| local                 | -                    | false                                                                   | The controller runs outside of a cluster and the webhook will not be started                                                                    |
| insecure              | -                    | false                                                                   | Enable insecure mode. The test machinery runs in insecure mode which means that local testdefs are allowed and therefore hostPaths are mounted. |