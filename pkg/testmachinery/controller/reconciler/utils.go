// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

func (r *TestmachineryReconciler) getImagePullSecrets() []string {
	imagePullSecrets := testmachinery.GetConfig().ImagePullSecretNames
	return imagePullSecrets
}

// RetryTimeoutExceeded returns whether the retry timeout is exceeded or not.
func RetryTimeoutExceeded(tr *tmv1beta1.Testrun) bool {
	timeout := testmachinery.GetRetryTimeout()
	passedTime := time.Since(tr.CreationTimestamp.Time)
	return passedTime.Milliseconds() >= timeout.Milliseconds()
}
