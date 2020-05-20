package shootflavors

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

func SetupWorker(cloudprofile gardencorev1beta1.CloudProfile, workers []gardencorev1beta1.Worker) ([]gardencorev1beta1.Worker, error) {
	res := make([]gardencorev1beta1.Worker, len(workers))
	for i, w := range workers {
		worker := w.DeepCopy()
		if worker.Machine.Image != nil && (worker.Machine.Image.Version == nil || *worker.Machine.Image.Version == common.PatternLatest) {
			version, err := util.GetLatestMachineImageVersion(cloudprofile, worker.Machine.Image.Name)
			if err != nil {
				return nil, err
			}
			worker.Machine.Image.Version = &version.Version
		}
		res[i] = *worker
	}
	return res, nil
}
