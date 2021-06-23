// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package reconciler

import (
	"context"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

func (r *TestmachineryReconciler) getImagePullSecrets(ctx context.Context) []string {
	imagePullSecrets := testmachinery.GetConfig().ImagePullSecretNames
	return imagePullSecrets
}

// RetryTimeoutExceeded returns whether the retry timeout is exceeded or not.
func RetryTimeoutExceeded(tr *tmv1beta1.Testrun) bool {
	timeout := testmachinery.GetRetryTimeout()
	passedTime := time.Since(tr.CreationTimestamp.Time)
	return passedTime.Milliseconds() >= timeout.Milliseconds()
}
