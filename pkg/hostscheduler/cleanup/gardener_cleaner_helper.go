//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cleanup

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/gardener/gardener/pkg/operation/common"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/gardener/gardener/pkg/operation/botanist"
	"github.com/gardener/gardener/pkg/utils/flow"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

var (
	// NotMonitoringComponent is a requirement that something doesn't have the GardenRole GardenRoleMonitoring.
	NotMonitoringComponent = botanist.MustNewRequirement(common.GardenRole, selection.NotEquals, common.GardenRoleMonitoring)

	// NotAddonManagerReconcile is a requirement that something doesnt have the label addonmanager.kubernetes.io/mode = Reconcile
	NotAddonManagerReconcile = botanist.MustNewRequirement("addonmanager.kubernetes.io/mode", selection.NotEquals, "Reconcile")

	// NamespaceCleanOptions is the delete selector for Namespaces.
	NamespaceCleanOptions = botanist.ListOptions(client.UseListOptions(&client.ListOptions{
		LabelSelector: botanist.CleanupSelector,
		FieldSelector: fields.AndSelectors(
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespacePublic),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespaceSystem),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespaceDefault),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, corev1.NamespaceNodeLease),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, "garden-setup-state"),
		),
	}))

	// NotMonitoringSelector is a selector that excludes monitoring and addon components.
	NotMonitoringOrAddonSelector = labels.NewSelector().Add(NotMonitoringComponent, NotAddonManagerReconcile)

	SecretsCleanOptions = botanist.ListOptions(client.UseListOptions(&botanist.NoCleanupPreventionListOptions))

	ConfigMapCleanOptions = botanist.ListOptions(client.UseListOptions(&botanist.NoCleanupPreventionListOptions))
)

func cleanResourceFn(logger logr.Logger, c client.Client, list runtime.Object, t string, finalize bool, opts ...botanist.CleanOptionFunc) flow.TaskFn {
	timeout := 3 * time.Minute
	logCleaner(logger, c, list, t, opts...)
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

// CleanWebhooks deletes all Webhooks in the Shoot cluster that are not being managed by the addon manager.
func CleanWebhooks(ctx context.Context, l logr.Logger, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(l, c, &admissionregistrationv1beta1.MutatingWebhookConfigurationList{}, "MutationsWebhook", true, addMonitoringOrAddonListOptions(botanist.MutatingWebhookConfigurationCleanOptions)),
		cleanResourceFn(l, c, &admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}, "ValidationWebhook", true, addMonitoringOrAddonListOptions(botanist.ValidatingWebhookConfigurationCleanOptions)),
	)(ctx)
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func CleanExtendedAPIs(ctx context.Context, l logr.Logger, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(l, c, &apiextensionsv1beta1.CustomResourceDefinitionList{}, "CRD", true, addMonitoringOrAddonListOptions(botanist.CustomResourceDefinitionCleanOptions)),
	)(ctx)
}

// todo : secrets configmaps

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, l logr.Logger, c client.Client) error {
	return flow.Parallel(
		cleanResourceFn(l, c, &batchv1beta1.CronJobList{}, "CronJob", false, addMonitoringOrAddonListOptions(botanist.CronJobCleanOptions)),
		cleanResourceFn(l, c, &appsv1.DaemonSetList{}, "DaemonSet", false, addMonitoringOrAddonListOptions(botanist.DaemonSetCleanOptions)),
		cleanResourceFn(l, c, &appsv1.DeploymentList{}, "Deployment", false, addMonitoringOrAddonListOptions(botanist.DeploymentCleanOptions)),
		cleanResourceFn(l, c, &batchv1.JobList{}, "Job", false, addMonitoringOrAddonListOptions(botanist.JobCleanOptions)),
		cleanResourceFn(l, c, &corev1.PodList{}, "Pod", false, addMonitoringOrAddonListOptions(botanist.PodCleanOptions)),
		cleanResourceFn(l, c, &appsv1.ReplicaSetList{}, "ReplicaSet", false, addMonitoringOrAddonListOptions(botanist.ReplicaSetCleanOptions)),
		cleanResourceFn(l, c, &corev1.ReplicationControllerList{}, "ReplicationController", false, addMonitoringOrAddonListOptions(botanist.ReplicationControllerCleanOptions)),
		cleanResourceFn(l, c, &appsv1.StatefulSetList{}, "StatefulSet", false, addMonitoringOrAddonListOptions(botanist.StatefulSetCleanOptions)),
		cleanResourceFn(l, c, &corev1.PersistentVolumeClaimList{}, "PVC", false, addMonitoringOrAddonListOptions(botanist.PersistentVolumeClaimCleanOptions)),
		cleanResourceFn(l, c, &extensionsv1beta1.IngressList{}, "Ingress", false, addMonitoringOrAddonListOptions(botanist.IngressCleanOptions)),
		cleanResourceFn(l, c, &corev1.ServiceList{}, "Service", false, addMonitoringOrAddonListOptions(botanist.ServiceCleanOptions)),
		cleanResourceFn(l, c, &corev1.NamespaceList{}, "Namespace", false, NamespaceCleanOptions),
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

func logCleaner(logger logr.Logger, c client.Client, list runtime.Object, t string, opts ...botanist.CleanOptionFunc) {
	cleanOptions := &botanist.CleanOptions{}
	cleanOptions.ApplyOptions(opts)
	err := c.List(context.TODO(), list, cleanOptions.ListOpts...)
	if err != nil {
		logger.Error(err, "unable to list objects: %s")
		return
	}

	logger.Info(fmt.Sprintf("found %d list items", meta.LenList(list)))

	err = meta.EachListItem(list, func(obj runtime.Object) error {
		o, err := meta.Accessor(obj)
		if err != nil {
			logger.V(3).Info(err.Error())
			return nil
		}
		logger.Info(fmt.Sprintf("Found Type: %s Name: %s, Namespace: %s to delete", t, o.GetName(), o.GetNamespace()))
		return nil
	})
	if err != nil {
		logger.Error(err, "unable to list objects")
		return
	}
}
