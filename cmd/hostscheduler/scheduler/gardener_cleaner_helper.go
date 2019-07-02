package scheduler

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

var (
	// NotMonitoringComponent is a requirement that something doesn't have the GardenRole GardenRoleMonitoring.
	NotMonitoringComponent = botanist.MustNewRequirement(common.GardenRole, selection.NotEquals, common.GardenRoleMonitoring)

	// NotAddonManagerReconcile is a requirment that something doesnt have the label addonmanager.kubernetes.io/mode = Reconcile
	NotAddonManagerReconcile = botanist.MustNewRequirement("addonmanager.kubernetes.io/mode", selection.NotEquals, "Reconcile")

	// NotMonitoringSelector is a selector that excludes monitoring and addon components.
	NotMonitoringOrAddonSelector = labels.NewSelector().Add(NotMonitoringComponent, NotAddonManagerReconcile)

	// NotSystemComponentListOptions are ListOptions that exclude system components.
	NotMonitoringOrAddonListOptions = client.ListOptions{
		LabelSelector: NotMonitoringOrAddonSelector,
	}
)

func cleanResourceFn(c client.Client, list runtime.Object, t string, finalize bool, opts ...botanist.CleanOptionFunc) flow.TaskFn {
	timeout := 3 * time.Minute
	logCleaner(c, list, t, opts...)
	mkCleaner := func(finalize bool) flow.TaskFn {
		newOpts := make([]botanist.CleanOptionFunc, len(opts), len(opts)+1)
		copy(newOpts, opts)

		if !finalize {
			newOpts = append(newOpts, botanist.DeleteOptions(client.GracePeriodSeconds(0)))
		} else {
			newOpts = append(newOpts, botanist.Finalize)
		}

		return func(ctx context.Context) error {
			return botanist.CleanMatching(ctx, c, list, newOpts...)
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
		cleanResourceFn(c, &apiregistrationv1beta1.APIServiceList{}, "ApiServices", true, addMonitoringOrAddonListOptions(botanist.APIServiceCleanOptions)),
		cleanResourceFn(c, &apiextensionsv1beta1.CustomResourceDefinitionList{}, "CRD", true, addMonitoringOrAddonListOptions(botanist.CustomResourceDefinitionCleanOptions)),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(c, &batchv1beta1.CronJobList{}, "CronJob", false, addMonitoringOrAddonListOptions(botanist.CronJobCleanOptions)),
		cleanResourceFn(c, &appsv1.DaemonSetList{}, "DaemonSet", false, addMonitoringOrAddonListOptions(botanist.DaemonSetCleanOptions)),
		cleanResourceFn(c, &appsv1.DeploymentList{}, "Deployment", false, addMonitoringOrAddonListOptions(botanist.DeploymentCleanOptions)),
		cleanResourceFn(c, &batchv1.JobList{}, "Job", false, addMonitoringOrAddonListOptions(botanist.JobCleanOptions)),
		cleanResourceFn(c, &corev1.PodList{}, "Pod", false, addMonitoringOrAddonListOptions(botanist.PodCleanOptions)),
		cleanResourceFn(c, &appsv1.ReplicaSetList{}, "ReplicaSet", false, addMonitoringOrAddonListOptions(botanist.ReplicaSetCleanOptions)),
		cleanResourceFn(c, &corev1.ReplicationControllerList{}, "ReplicationController", false, addMonitoringOrAddonListOptions(botanist.ReplicationControllerCleanOptions)),
		cleanResourceFn(c, &appsv1.StatefulSetList{}, "StatefulSet", false, addMonitoringOrAddonListOptions(botanist.StatefulSetCleanOptions)),
		cleanResourceFn(c, &corev1.PersistentVolumeClaimList{}, "PVC", false, addMonitoringOrAddonListOptions(botanist.PersistentVolumeClaimCleanOptions)),
		cleanResourceFn(c, &extensionsv1beta1.IngressList{}, "Ingress", false, addMonitoringOrAddonListOptions(botanist.IngressCleanOptions)),
		cleanResourceFn(c, &corev1.ServiceList{}, "Service", false, addMonitoringOrAddonListOptions(botanist.ServiceCleanOptions)),
		cleanResourceFn(c, &corev1.NamespaceList{}, "Namespace", false, botanist.NamespaceCleanOptions),
	)(ctx)
}

func addMonitoringOrAddonListOptions(f botanist.CleanOptionFunc) botanist.CleanOptionFunc {
	listOpts := &client.ListOptions{}
	cleanOptions := &botanist.CleanOptions{}
	f(cleanOptions)
	if len(cleanOptions.ListOpts) == 0 {
		return f
	}
	cleanOptions.ListOpts[0](listOpts)
	listOpts.LabelSelector.Add(NotMonitoringComponent, NotAddonManagerReconcile)
	return botanist.ListOptions(client.UseListOptions(listOpts))
}

func logCleaner(c client.Client, list runtime.Object, t string, opts ...botanist.CleanOptionFunc) {
	cleanOptions := &botanist.CleanOptions{}
	cleanOptions.ApplyOptions(opts)
	err := c.List(context.TODO(), list, cleanOptions.ListOpts...)
	if err != nil {
		log.Warnf("unable to list objects: %s", err.Error())
		return
	}

	log.Debugf("found %d list items", meta.LenList(list))

	err = meta.EachListItem(list, func(obj runtime.Object) error {
		o, err := meta.Accessor(obj)
		if err != nil {
			log.Debug(err)
			return nil
		}
		log.Infof("Found Type: %s Name: %s, Namespace: %s to delete", t, o.GetName(), o.GetNamespace())
		return nil
	})
	if err != nil {
		log.Warnf("unable to list objects: %s", err.Error())
		return
	}
}
