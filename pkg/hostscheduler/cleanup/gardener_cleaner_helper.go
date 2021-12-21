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
	"time"

	netv1 "k8s.io/api/networking/v1"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/apimachinery/pkg/fields"

	"k8s.io/apimachinery/pkg/selection"

	"github.com/gardener/gardener/pkg/utils/flow"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Provider is the kubernetes provider label.
	Provider = "provider"
	// KubernetesProvider is the 'kubernetes' value of the Provider label.
	KubernetesProvider = "kubernetes"
	// MetadataNameField ist the `metadata.name` field for a field selector.
	MetadataNameField = "metadata.name"
)

const (
	// ShootNoCleanup is a constant for a label on a resource indicating that the Gardener cleaner should not delete this
	// resource when cleaning a shoot during the deletion flow.
	ShootNoCleanup = "shoot.gardener.cloud/no-cleanup"
)

var (

	// NotSystemComponent is a requirement that something doesn't have the GardenRole GardenRoleSystemComponent.
	NotSystemComponent = utils.MustNewRequirement(v1beta1constants.GardenRole, selection.NotEquals, v1beta1constants.GardenRoleSystemComponent)

	// NoCleanupPrevention is a requirement that the ShootNoCleanup label of something is not true.
	NoCleanupPrevention = utils.MustNewRequirement(ShootNoCleanup, selection.NotEquals, "true")

	// NoCleanupPreventionListOption are CollectionMatching that exclude system components or non-auto cleaned up resource.
	NoCleanupPreventionListOption = client.MatchingLabelsSelector{Selector: CleanupSelector}

	// NotAddonManagerReconcile is a requirement that something doesnt have the label addonmanager.kubernetes.io/mode = Reconcile
	NotAddonManagerReconcile = MustNewRequirement("addonmanager.kubernetes.io/mode", selection.NotEquals, "Reconcile")

	// NotKubernetesClusterService is a requirement that something doesnt have the label kubernetes.io/cluster-service = true
	NotKubernetesClusterService = MustNewRequirement("kubernetes.io/cluster-service", selection.NotEquals, "true")

	// NotKubernetesProvider is a requirement that the Provider label of something is not KubernetesProvider.
	NotKubernetesProvider = utils.MustNewRequirement(Provider, selection.NotEquals, KubernetesProvider)

	// CleanupSelector is a selector that excludes system components and all resources not considered for auto cleanup.
	CleanupSelector = labels.NewSelector().Add(NotSystemComponent).Add(NoCleanupPrevention)

	// NamespaceCleanOption is the delete selector for Namespaces that excludes system namespaces.
	NamespaceCleanOption = []client.ListOption{
		client.MatchingLabelsSelector{Selector: CleanupSelector},
		client.MatchingFieldsSelector{
			Selector: fields.AndSelectors(
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespacePublic),
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespaceSystem),
				fields.OneTermNotEqualSelector(MetadataNameField, metav1.NamespaceDefault),
				fields.OneTermNotEqualSelector(MetadataNameField, corev1.NamespaceNodeLease),
			),
		},
	}

	// DaemonSetCleanOption is the delete selector for DaemonSets.
	DaemonSetCleanOption = &NoCleanupPreventionListOption

	// DeploymentCleanOption is the delete selector for Deployments.
	DeploymentCleanOption = &NoCleanupPreventionListOption

	// StatefulSetCleanOption is the delete selector for StatefulSets.
	StatefulSetCleanOption = &NoCleanupPreventionListOption

	// ServiceCleanOption is the delete selector for Services.
	ServiceCleanOption = client.MatchingLabelsSelector{
		Selector: labels.NewSelector().Add(NotKubernetesProvider, NotSystemComponent, NoCleanupPrevention),
	}

	// MutatingWebhookConfigurationCleanOption is the delete selector for MutatingWebhookConfigurations.
	MutatingWebhookConfigurationCleanOption = &NoCleanupPreventionListOption

	// ValidatingWebhookConfigurationCleanOption is the delete selector for ValidatingWebhookConfigurations.
	ValidatingWebhookConfigurationCleanOption = &NoCleanupPreventionListOption

	// CustomResourceDefinitionCleanOption is the delete selector for CustomResources.
	CustomResourceDefinitionCleanOption = &NoCleanupPreventionListOption

	// CronJobCleanOption is the delete selector for CronJobs.
	CronJobCleanOption = &NoCleanupPreventionListOption

	// IngressCleanOption is the delete selector for Ingresses.
	IngressCleanOption = &NoCleanupPreventionListOption

	// JobCleanOption is the delete selector for Jobs.
	JobCleanOption = &NoCleanupPreventionListOption

	// PodCleanOption is the delete selector for Pods.
	PodCleanOption = &NoCleanupPreventionListOption

	// ReplicaSetCleanOption is the delete selector for ReplicaSets.
	ReplicaSetCleanOption = &NoCleanupPreventionListOption

	// ReplicationControllerCleanOption is the delete selector for ReplicationControllers.
	ReplicationControllerCleanOption = &NoCleanupPreventionListOption

	// PersistentVolumeClaimCleanOption is the delete selector for PersistentVolumeClaims.
	PersistentVolumeClaimCleanOption = &NoCleanupPreventionListOption
)

func cleanResourceFn(ctx context.Context, logger logr.Logger, c client.Client, list client.ObjectList, t string, finalize bool, opts ...client.ListOption) flow.TaskFn {
	logCleaner(ctx, logger, c, list, t, opts...)

	return func(ctx context.Context) error {
		return retry.Until(ctx, 5*time.Second, func(ctx context.Context) (done bool, err error) {

			if err := c.List(ctx, list, opts...); err != nil {
				return retry.MinorError(err)
			}

			foundObjects := make([]client.Object, 0)
			err = meta.EachListItem(list, func(obj runtime.Object) error {
				foundObjects = append(foundObjects, obj.(client.Object))
				return nil
			})
			if err != nil {
				return retry.SevereError(err)
			}

			if len(foundObjects) == 0 {
				return retry.Ok()
			}

			for _, obj := range foundObjects {
				if err := c.Delete(ctx, obj); err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					return retry.MinorError(err)
				}
			}

			return retry.MinorError(errors.New("still objects left"))
		})
	}
}

// CleanWebhooks deletes all Webhooks in the Shoot cluster that are not being managed by the addon manager.
func CleanWebhooks(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	sel := labels.NewSelector()
	sel.Add(requirements...)

	return flow.Parallel(
		cleanResourceFn(ctx, l, c, &admissionregistrationv1beta1.MutatingWebhookConfigurationList{}, "MutationsWebhook", true, MutatingWebhookConfigurationCleanOption, client.MatchingLabelsSelector{Selector: sel}),
		cleanResourceFn(ctx, l, c, &admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}, "ValidationWebhook", true, ValidatingWebhookConfigurationCleanOption, client.MatchingLabelsSelector{Selector: sel}),
	)(ctx)
}

// CleanExtendedAPIs removes API extensions like CRDs and API services from the Shoot cluster.
func CleanExtendedAPIs(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	sel := labels.NewSelector()
	sel.Add(requirements...)

	return flow.Parallel(
		cleanResourceFn(ctx, l, c, &apiextensionsv1beta1.CustomResourceDefinitionList{}, "CRD", true, CustomResourceDefinitionCleanOption, client.MatchingLabelsSelector{Selector: sel}),
	)(ctx)
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func CleanKubernetesResources(ctx context.Context, l logr.Logger, c client.Client, requirements labels.Requirements) error {
	sel := labels.NewSelector()
	sel.Add(requirements...)
	labelOption := client.MatchingLabelsSelector{Selector: sel}
	return flow.Parallel(
		cleanResourceFn(ctx, l, c, &batchv1beta1.CronJobList{}, "CronJob", false, CronJobCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &appsv1.DaemonSetList{}, "DaemonSet", false, DaemonSetCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &appsv1.DeploymentList{}, "Deployment", false, DeploymentCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &batchv1.JobList{}, "Job", false, JobCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &corev1.PodList{}, "Pod", false, PodCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &appsv1.ReplicaSetList{}, "ReplicaSet", false, ReplicaSetCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &corev1.ReplicationControllerList{}, "ReplicationController", false, ReplicationControllerCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &appsv1.StatefulSetList{}, "StatefulSet", false, StatefulSetCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &corev1.PersistentVolumeClaimList{}, "PVC", false, PersistentVolumeClaimCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &netv1.IngressList{}, "Ingress", false, IngressCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &corev1.ServiceList{}, "Service", false, ServiceCleanOption, labelOption),
		cleanResourceFn(ctx, l, c, &corev1.NamespaceList{}, "Namespace", false, NamespaceCleanOption...),
	)(ctx)
}

func logCleaner(ctx context.Context, logger logr.Logger, c client.Client, list client.ObjectList, t string, opts ...client.ListOption) {
	err := c.List(ctx, list, opts...)
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

func MustNewRequirement(key string, op selection.Operator, val string) labels.Requirement {
	req, err := labels.NewRequirement(key, op, []string{val})
	utilruntime.Must(err)
	return *req
}
