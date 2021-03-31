package garbagecollection

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/test-infra/pkg/util/s3"

	argov1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testmachinery"
)

// GCWorkflowArtifacts collects all outputs of a workflow by traversing through nodes and collect outputs artifacts from the s3 storage.
// These artifacts are then deleted form the s3 storage.
func GCWorkflowArtifacts(log logr.Logger, s3Client s3.Client, wf *argov1.Workflow) (reconcile.Result, error) {
	if s3Client == nil {
		log.V(3).Info("skip garbage collection of artifacts")
		return reconcile.Result{}, nil
	}
	for _, node := range wf.Status.Nodes {
		if node.Outputs == nil {
			continue
		}
		for _, artifact := range node.Outputs.Artifacts {
			log.V(5).Info(fmt.Sprintf("Processing artifact %s", artifact.Name))
			if artifact.S3 != nil {
				err := s3Client.RemoveObject("", artifact.S3.Key)
				if err != nil {
					log.Error(err, "unable to delete object from object storage", "artifact", artifact.S3.Key)

					// do not retry deletion if the key does not not exist in s3 anymore
					// maybe use const from aws lib -> need to change to aws lib
					if err.Error() != "The specified key does not exist." {
						return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
					}
				}
				log.V(5).Info("object deleted", "artifact", artifact.S3.Key)
			}
		}
	}

	return reconcile.Result{}, nil
}

// CleanWorkflowPods deletes all pods of a completed workflow.
// cleanup pods to remove workload from the api server and etcd.
// logs are still accessible through "archiveLogs" option in argo
func CleanWorkflowPods(c client.Client, wf *argov1.Workflow) error {
	var result *multierror.Error
	if testmachinery.CleanWorkflowPods() {
		for nodeName, node := range wf.Status.Nodes {
			if node.Type == argov1.NodeTypePod {
				if err := deletePod(c, testmachinery.GetNamespace(), nodeName); err != nil {
					result = multierror.Append(result, fmt.Errorf("unable delete pod %s: %s", nodeName, err.Error()))
				}
			}
		}
	}
	return result.ErrorOrNil()
}

func deletePod(c client.Client, namespace, name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return c.Delete(context.TODO(), pod)
}
