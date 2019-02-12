# Testmachinery


## Controller

| Name                | default                                                                 | Description                                                                                               |
| ------------------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| TM_NAMESPACE        | "default"                                                               | Namespace where the testmachinery-controller runs                                                         |
| PREPARE_IMAGE       | "eu.gcr.io/gardener-project/gardener/testmachinery/prepare-step:0.28.0" | Image that is used in the prepare step.                                                                   |
| BASE_IMAGE          | "default"                                                               | Default image for test defintion with no explicit image.                                                  |
| CLEAN_WORKFLOW_PODS | "false"                                                                 | Deletes all pods when a workflow is finished. Logs are still accessible.                                  |
| S3_ENDPOINT         | ""                                                                      | Endpoint url of the minio/s3 instance where argo stores it's artifacts (e.g. minio-service.deafult:9000). |
| S3_ACCESS_KEY       | ""                                                                      | Minio/S3 access key                                                                                       |
| S3_SECRET_KEY       | ""                                                                      | Minio/S3 secret key                                                                                       |
| S3_BUCKET_NAME      | ""                                                                      | Minio/S3 bucket name, where argo stores it's artifacts.                                                   |
| GIT_SECRETS         | ""                                                                      | Github configuration with technical users and endpoint information.                                       |
