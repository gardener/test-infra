package scheduler

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/operation/common"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

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

var (
	// NotMonitoringComponent is a requirement that something doesn't have the GardenRole GardenRoleMonitoring.
	NotMonitoringComponent = botanist.MustNewRequirement(common.GardenRole, selection.NotEquals, common.GardenRoleMonitoring)

	// NotSystemComponentSelector is a selector that excludes system components.
	NotSystemComponentAndMonitoringSelector = labels.NewSelector().Add(botanist.NotSystemComponent, NotMonitoringComponent)

	// NotSystemComponentListOptions are ListOptions that exclude system components.
	NotSystemComponentListOptions = client.ListOptions{
		LabelSelector: NotSystemComponentAndMonitoringSelector,
	}

	// CustomResourceDefinitionDeleteSelector is the delete selector for CustomResources.
	CustomResourceDefinitionDeleteSelector = &NotSystemComponentListOptions
	// CustomResourceDefinitionCheckSelector is the check selector for CustomResources.
	CustomResourceDefinitionCheckSelector = CustomResourceDefinitionDeleteSelector

	// DaemonSetDeleteSelector is the delete selector for DaemonSets.
	DaemonSetDeleteSelector = &NotSystemComponentListOptions
	// DaemonSetCheckSelector is the check selector for DaemonSets.
	DaemonSetCheckSelector = DaemonSetDeleteSelector

	// DeploymentDeleteSelector is the delete selector for Deployments.
	DeploymentDeleteSelector = &NotSystemComponentListOptions
	// DeploymentCheckSelector is the check selector for Deployments.
	DeploymentCheckSelector = DeploymentDeleteSelector

	// StatefulSetDeleteSelector is the delete selector for StatefulSets.
	StatefulSetDeleteSelector = &NotSystemComponentListOptions
	// StatefulSetCheckSelector is the check selector for StatefulSets.
	StatefulSetCheckSelector = StatefulSetDeleteSelector

	// CronJobDeleteSelector is the delete selector for CronJobs.
	CronJobDeleteSelector = &NotSystemComponentListOptions
	// CronJobCheckSelector is the check selector for CronJobs.
	CronJobCheckSelector = CronJobDeleteSelector

	// IngressDeleteSelector is the delete selector for Ingresses.
	IngressDeleteSelector = &NotSystemComponentListOptions
	// IngressCheckSelector is the check selector for Ingresses.
	IngressCheckSelector = IngressDeleteSelector

	// JobDeleteSelector is the delete selector for Jobs.
	JobDeleteSelector = &NotSystemComponentListOptions
	// JobCheckSelector is the check selector for Jobs.
	JobCheckSelector = JobDeleteSelector

	// PodDeleteSelector is the delete selector for Pods.
	PodDeleteSelector = &NotSystemComponentListOptions
	// PodCheckSelector is the check selector for Pods.
	PodCheckSelector = PodDeleteSelector

	// ReplicaSetDeleteSelector is the delete selector for ReplicaSets.
	ReplicaSetDeleteSelector = &NotSystemComponentListOptions
	// ReplicaSetCheckSelector is the check selector for ReplicaSets.
	ReplicaSetCheckSelector = ReplicaSetDeleteSelector

	// ReplicationControllerDeleteSelector is the delete selector for ReplicationControllers.
	ReplicationControllerDeleteSelector = &NotSystemComponentListOptions
	// ReplicationControllerCheckSelector is the check selector for ReplicationControllers.
	ReplicationControllerCheckSelector = ReplicationControllerDeleteSelector

	// PersistentVolumeClaimDeleteSelector is the delete selector for PersistentVolumeClaims.
	PersistentVolumeClaimDeleteSelector = &NotSystemComponentListOptions
	// PersistentVolumeClaimCheckSelector is the check selector for PersistentVolumeClaims.
	PersistentVolumeClaimCheckSelector = PersistentVolumeClaimDeleteSelector
)

func cleanResourceFn(c client.Client, deleteSelector, checkSelector *client.ListOptions, list runtime.Object, finalize bool) flow.TaskFn {
	timeout := 3 * time.Minute
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
		return mkCleaner(false).RetryUntilTimeout(5*time.Second, timeout)
	}

	return func(ctx context.Context) error {
		return mkCleaner(false).RetryUntilTimeout(5*time.Second, timeout).Recover(mkCleaner(true).RetryUntilTimeout(5*time.Second, timeout).ToRecoverFn())(ctx)
	}
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func CleanExtendedAPIs(ctx context.Context, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(c, botanist.APIServiceDeleteSelector, botanist.APIServiceCheckSelector, &apiregistrationv1beta1.APIServiceList{}, true),
		cleanResourceFn(c, CustomResourceDefinitionDeleteSelector, CustomResourceDefinitionCheckSelector, &apiextensionsv1beta1.CustomResourceDefinitionList{}, true),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(c, CronJobDeleteSelector, CronJobCheckSelector, &batchv1beta1.CronJobList{}, false),
		cleanResourceFn(c, DaemonSetDeleteSelector, DaemonSetCheckSelector, &appsv1.DaemonSetList{}, false),
		cleanResourceFn(c, DeploymentDeleteSelector, DeploymentCheckSelector, &appsv1.DeploymentList{}, false),
		cleanResourceFn(c, JobDeleteSelector, JobCheckSelector, &batchv1.JobList{}, false),
		cleanResourceFn(c, PodDeleteSelector, PodCheckSelector, &corev1.PodList{}, false),
		cleanResourceFn(c, ReplicaSetDeleteSelector, ReplicaSetCheckSelector, &appsv1.ReplicaSetList{}, false),
		cleanResourceFn(c, ReplicationControllerDeleteSelector, ReplicationControllerCheckSelector, &corev1.ReplicationControllerList{}, false),
		cleanResourceFn(c, StatefulSetDeleteSelector, StatefulSetCheckSelector, &appsv1.StatefulSetList{}, false),
		cleanResourceFn(c, PersistentVolumeClaimDeleteSelector, PersistentVolumeClaimCheckSelector, &corev1.PersistentVolumeClaimList{}, false),
		cleanResourceFn(c, IngressDeleteSelector, IngressCheckSelector, &extensionsv1beta1.IngressList{}, false),
		cleanResourceFn(c, botanist.ServiceDeleteSelector, botanist.ServiceCheckSelector, &corev1.ServiceList{}, false),
		cleanResourceFn(c, botanist.NamespaceDeleteSelector, botanist.NamespaceCheckSelector, &corev1.NamespaceList{}, false),
	)(ctx)
}
