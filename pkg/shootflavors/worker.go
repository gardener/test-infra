package shootflavors

import (
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

func SetupWorker(k8sClient kubernetes.Interface, workers []gardenv1alpha1.Worker, cloudprofile string) ([]gardenv1alpha1.Worker, error) {
	res := make([]gardenv1alpha1.Worker, len(workers))
	for i, w := range workers {
		worker := w.DeepCopy()
		if worker.Machine.Image != nil && worker.Machine.Image.Version == common.PatternLatest {
			version, err := util.GetLatestMachineImageVersion(k8sClient, cloudprofile, worker.Machine.Image.Name)
			if err != nil {
				return nil, err
			}
			worker.Machine.Image.Version = version.Version
		}
		res[i] = *worker
	}
	return res, nil
}
