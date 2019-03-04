export S3_ENDPOINT=$(minikube service minio-service --url | xargs basename)
export S3_ACCESS_KEY=$(cat charts/bootstrap_tm_prerequisites/values.yaml | yq r - 'objectStorage.secret.accessKey')
export S3_SECRET_KEY=$(cat charts/bootstrap_tm_prerequisites/values.yaml | yq r - 'objectStorage.secret.secretKey')