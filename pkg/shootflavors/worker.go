package shootflavors

import (
	"encoding/json"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

func SetupWorker(cloudprofile gardencorev1beta1.CloudProfile, workers []gardencorev1beta1.Worker) ([]gardencorev1beta1.Worker, error) {
	res := make([]gardencorev1beta1.Worker, len(workers))
	for i, w := range workers {
		worker := w.DeepCopy()
		if worker.Machine.Image != nil && (worker.Machine.Image.Version == nil || *worker.Machine.Image.Version == common.PatternLatest) {
			version, err := util.GetLatestMachineImageVersion(cloudprofile, worker.Machine.Image.Name, *worker.Machine.Architecture)
			if err != nil {
				return nil, err
			}
			worker.Machine.Image.Version = &version.Version
		}

		if cloudprofile.Spec.Type == "aws" {
			providerConfig := v1alpha1.WorkerConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       "WorkerConfig",
					APIVersion: "aws.provider.extensions.gardener.cloud/v1alpha1",
				},
				InstanceMetadataOptions: &v1alpha1.InstanceMetadataOptions{
					HTTPTokens:              ptr.To(v1alpha1.HTTPTokensRequired),
					HTTPPutResponseHopLimit: ptr.To(int64(2)),
				},
			}
			js, err := json.Marshal(providerConfig)
			if err != nil {
				return nil, err
			}
			worker.ProviderConfig = &runtime.RawExtension{
				Raw: js,
			}
		}
		res[i] = *worker
	}
	return res, nil
}
