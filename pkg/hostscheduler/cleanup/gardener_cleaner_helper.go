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

	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/apimachinery/pkg/selection"

	"github.com/gardener/gardener/pkg/operation/botanist"
	"github.com/gardener/gardener/pkg/utils/flow"
	utilclient "github.com/gardener/gardener/pkg/utils/kubernetes/client"
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
	// NotAddonManagerReconcile is a requirement that something doesnt have the label addonmanager.kubernetes.io/mode = Reconcile
	NotAddonManagerReconcile = botanist.MustNewRequirement("addonmanager.kubernetes.io/mode", selection.NotEquals, "Reconcile")

	// NotKubernetesClusterService is a requirement that something doesnt have the label kubernetes.io/cluster-service = true
	NotKubernetesClusterService = botanist.MustNewRequirement("kubernetes.io/cluster-service", selection.NotEquals, "true")

	// NamespaceCleanOptions is the delete selector for Namespaces.
	NamespaceCleanOptions = utilclient.DeleteWith(utilclient.CollectionMatching(client.UseListOptions(&client.ListOptions{
		LabelSelector: botanist.CleanupSelector,
		FieldSelector: fields.AndSelectors(
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespacePublic),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespaceSystem),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, metav1.NamespaceDefault),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, corev1.NamespaceNodeLease),
			fields.OneTermNotEqualSelector(botanist.MetadataNameField, "garden-setup-state"),
		),
	})))
)

func cleanResourceFn(logger logr.Logger, cleanOps utilclient.CleanOps, c client.Client, list runtime.Object, t string, finalize bool, opts ...utilclient.CleanOptionFunc) flow.TaskFn {
	logCleaner(logger, c, list, t, opts...)

	return func(ctx context.Context) error {
		return retry.Until(ctx, botanist.DefaultInterval, func(ctx context.Context) (done bool, err error) {
			if err := cleanOps.CleanAndEnsureGone(ctx, c, list, opts...); err != nil {
				if utilclient.AreObjectsRemaining(err) {
					return retry.MinorError(err)
				}
				return retry.SevereError(err)
			}
			return retry.Ok()
		})
	}
}

// CleanWebhooks deletes all Webhooks in the Shoot cluster that are not being managed by the addon manager.
func CleanWebhooks(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	ops := utilclient.DefaultCleanOps()
	return flow.Parallel(
		cleanResourceFn(l, ops, c, &admissionregistrationv1beta1.MutatingWebhookConfigurationList{}, "MutationsWebhook", true, addAdditionalListOptions(botanist.MutatingWebhookConfigurationCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}, "ValidationWebhook", true, addAdditionalListOptions(botanist.ValidatingWebhookConfigurationCleanOptions, requirements)),
	)(ctx)
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func CleanExtendedAPIs(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	ops := utilclient.DefaultCleanOps()
	return flow.Parallel(
		cleanResourceFn(l, ops, c, &apiextensionsv1beta1.CustomResourceDefinitionList{}, "CRD", true, addAdditionalListOptions(botanist.CustomResourceDefinitionCleanOptions, requirements)),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	ops := utilclient.DefaultCleanOps()
	return flow.Parallel(
		cleanResourceFn(l, ops, c, &batchv1beta1.CronJobList{}, "CronJob", false, addAdditionalListOptions(botanist.CronJobCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &appsv1.DaemonSetList{}, "DaemonSet", false, addAdditionalListOptions(botanist.DaemonSetCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &appsv1.DeploymentList{}, "Deployment", false, addAdditionalListOptions(botanist.DeploymentCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &batchv1.JobList{}, "Job", false, addAdditionalListOptions(botanist.JobCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &corev1.PodList{}, "Pod", false, addAdditionalListOptions(botanist.PodCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &appsv1.ReplicaSetList{}, "ReplicaSet", false, addAdditionalListOptions(botanist.ReplicaSetCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &corev1.ReplicationControllerList{}, "ReplicationController", false, addAdditionalListOptions(botanist.ReplicationControllerCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &appsv1.StatefulSetList{}, "StatefulSet", false, addAdditionalListOptions(botanist.StatefulSetCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &corev1.PersistentVolumeClaimList{}, "PVC", false, addAdditionalListOptions(botanist.PersistentVolumeClaimCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &extensionsv1beta1.IngressList{}, "Ingress", false, addAdditionalListOptions(botanist.IngressCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &corev1.ServiceList{}, "Service", false, addAdditionalListOptions(botanist.ServiceCleanOptions, requirements)),
		cleanResourceFn(l, ops, c, &corev1.NamespaceList{}, "Namespace", false, NamespaceCleanOptions),
	)(ctx)
}

func addAdditionalListOptions(f utilclient.CleanOptionFunc, requirements labels.Requirements) utilclient.CleanOptionFunc {
	cleanOptions := &utilclient.CleanOptions{}
	f(cleanOptions)
	if cleanOptions.CollectionOptions == nil {
		return f
	}
	cleanOptions.CollectionOptions.LabelSelector = cleanOptions.CollectionOptions.LabelSelector.Add(NotAddonManagerReconcile, NotKubernetesClusterService)
	cleanOptions.CollectionOptions.LabelSelector = cleanOptions.CollectionOptions.LabelSelector.Add(requirements...)
	return utilclient.DeleteWith(utilclient.CollectionMatching(client.UseListOptions(cleanOptions.CollectionOptions)))
}

func logCleaner(logger logr.Logger, c client.Client, list runtime.Object, t string, opts ...utilclient.CleanOptionFunc) {
	cleanOptions := &utilclient.CleanOptions{}
	cleanOptions.ApplyOptions(opts)
	err := c.List(context.TODO(), list, client.UseListOptions(cleanOptions.CollectionOptions))
	if err != nil {
		logger.Error(err, "unable to list objects: %s")
		return
	}

	logger.V(3).Info(fmt.Sprintf("found %d list items", meta.LenList(list)))

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
