package scheduler

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/operation/botanist"
	"github.com/gardener/gardener/pkg/utils/flow"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

func cleanResourceFn(c client.Client, deleteSelector, checkSelector *client.ListOptions, list runtime.Object, finalize bool) flow.TaskFn {
	mkCleaner := func(finalize bool) flow.TaskFn {
		var opts []client.DeleteOptionFunc
		if !finalize {
			opts = []client.DeleteOptionFunc{client.GracePeriodSeconds(60)}
		} else {
			opts = []client.DeleteOptionFunc{client.GracePeriodSeconds(0)}
		}

		return func(ctx context.Context) error {
			return botanist.CleanMatching(ctx, c, deleteSelector, checkSelector, list, finalize, opts...)
		}
	}
	if !finalize {
		return mkCleaner(false).Retry(5 * time.Second)
	}

	return func(ctx context.Context) error {
		timeout := 3 * time.Minute
		return mkCleaner(false).RetryUntilTimeout(5*time.Second, timeout)(ctx)
	}
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func CleanExtendedAPIs(ctx context.Context, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(c, botanist.APIServiceDeleteSelector, botanist.APIServiceCheckSelector, &apiregistrationv1beta1.APIServiceList{}, true),
		cleanResourceFn(c, botanist.CustomResourceDefinitionDeleteSelector, botanist.CustomResourceDefinitionCheckSelector, &apiextensionsv1beta1.CustomResourceDefinitionList{}, true),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(c, botanist.CronJobDeleteSelector, botanist.CronJobCheckSelector, &batchv1beta1.CronJobList{}, false),
		cleanResourceFn(c, botanist.DaemonSetDeleteSelector, botanist.DaemonSetCheckSelector, &appsv1.DaemonSetList{}, false),
		cleanResourceFn(c, botanist.DeploymentDeleteSelector, botanist.DeploymentCheckSelector, &appsv1.DeploymentList{}, false),
		cleanResourceFn(c, botanist.IngressDeleteSelector, botanist.IngressCheckSelector, &extensionsv1beta1.IngressList{}, false),
		cleanResourceFn(c, botanist.JobDeleteSelector, botanist.JobCheckSelector, &batchv1.JobList{}, false),
		cleanResourceFn(c, botanist.NamespaceDeleteSelector, botanist.NamespaceCheckSelector, &corev1.NamespaceList{}, false),
		cleanResourceFn(c, botanist.PodDeleteSelector, botanist.PodCheckSelector, &corev1.PodList{}, false),
		cleanResourceFn(c, botanist.ReplicaSetDeleteSelector, botanist.ReplicaSetCheckSelector, &appsv1.ReplicaSetList{}, false),
		cleanResourceFn(c, botanist.ReplicationControllerDeleteSelector, botanist.ReplicationControllerCheckSelector, &corev1.ReplicationControllerList{}, false),
		cleanResourceFn(c, botanist.ServiceDeleteSelector, botanist.ServiceCheckSelector, &corev1.ServiceList{}, false),
		cleanResourceFn(c, botanist.StatefulSetDeleteSelector, botanist.StatefulSetCheckSelector, &appsv1.StatefulSetList{}, false),
		cleanResourceFn(c, botanist.PersistentVolumeClaimDeleteSelector, botanist.PersistentVolumeClaimCheckSelector, &corev1.PersistentVolumeClaimList{}, false),
	)(ctx)
}
