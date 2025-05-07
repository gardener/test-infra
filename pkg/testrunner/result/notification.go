// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"fmt"
	"os"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
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
	if err := os.WriteFile(notifyConfigFilePath, notifyConfig, 0600); err != nil {
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
		for _, step := range tr.Status.Steps {
			if step.Phase == argov1.NodeFailed {
				cfg.Email.MailBody = fmt.Sprintf("%s  - %s\n", cfg.Email.MailBody, step.TestDefinition.Name)
				cfg.Email.Recipients = append(cfg.Email.Recipients, step.TestDefinition.RecipientsOnFailure...)
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
