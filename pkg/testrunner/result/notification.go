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

package result

import (
	"fmt"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// GenerateNotificationConfigForAlerting creates a notification config file with email recipients if any test step has failed
// The config file is then evaluated by Concourse
func GenerateNotificationConfigForAlerting(tr []*tmv1beta1.Testrun, concourseOnErrorDir string) {
	if concourseOnErrorDir == "" {
		return
	}
	notifyConfig := createNotificationString(tr)
	if notifyConfig == nil {
		return
	}

	notifyConfigFilePath := fmt.Sprintf("%s/notify.cfg", concourseOnErrorDir)
	if err := writeToFile(notifyConfigFilePath, notifyConfig); err != nil {
		log.Warnf("Cannot write file email notification config to %s: %s", notifyConfigFilePath, err.Error())
		return
	}
	log.Infof("Successfully created file %s", notifyConfigFilePath)
}

func createNotificationString(testruns []*tmv1beta1.Testrun) []byte {

	cfg := notificationCfg{
		Email: email{
			Subject:  "Test Machinery - some steps failed",
			MailBody: "Test Machinery steps have failed.\n\nFailed Steps:\n",
		},
	}

	for _, tr := range testruns {
		cfg.Email.MailBody = fmt.Sprintf("%s  Testrun: %s\n", cfg.Email.MailBody, tr.Name)
		for _, steps := range tr.Status.Steps {
			for _, step := range steps {
				if step.Phase == argov1.NodeFailed {
					cfg.Email.MailBody = fmt.Sprintf("%s  - %s\n", cfg.Email.MailBody, step.TestDefinition.Name)
					for _, email := range step.TestDefinition.RecipientsOnFailure {
						cfg.Email.Recipients = append(cfg.Email.Recipients, email)
					}
				}
			}
		}
	}

	if len(cfg.Email.Recipients) != 0 {
		cfgBytes, err := yaml.Marshal(cfg)
		if err != nil {
			log.Warnf("Cannot encode email notification config %s", err.Error())
			return nil
		}
		return cfgBytes
	}
	return nil
}
