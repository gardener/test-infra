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

package pages

import (
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/auth"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/go-logr/logr"
	github2 "github.com/google/go-github/v27/github"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"time"
)

var PhaseIcon = map[v1alpha1.NodePhase]IconWithTooltip{
	v1beta1.PhaseStatusInit: {
		Icon:    "schedule",
		Tooltip: fmt.Sprintf("%s phase: Testrun is waiting to be scheduled", v1beta1.PhaseStatusInit),
		Color:   "grey",
	},
	v1beta1.PhaseStatusRunning: {
		Icon:    "autorenew",
		Tooltip: fmt.Sprintf("%s phase: Testrun is running", v1beta1.PhaseStatusRunning),
		Color:   "orange",
	},
	v1beta1.PhaseStatusSuccess: {
		Icon:    "done",
		Tooltip: fmt.Sprintf("%s phase: Testrun succeeded", v1beta1.PhaseStatusSuccess),
		Color:   "green",
	},
	v1beta1.PhaseStatusFailed: {
		Icon:    "clear",
		Tooltip: fmt.Sprintf("%s phase: Testrun failed", v1beta1.PhaseStatusFailed),
		Color:   "red",
	},
	v1beta1.PhaseStatusError: {
		Icon:    "clear",
		Tooltip: fmt.Sprintf("%s phase: Testrun errored", v1beta1.PhaseStatusError),
		Color:   "red",
	},
	v1beta1.PhaseStatusTimeout: {
		Icon:    "clear",
		Tooltip: fmt.Sprintf("%s phase: Testrun run longer than the specified timeout", v1beta1.PhaseStatusTimeout),
		Color:   "red",
	},
}

type runItem struct {
	Organization string
	Repository   string
	PR           int64

	Testrun  string
	Phase    IconWithTooltip
	Progress string

	ArgoURL string
}

type runDetailedItem struct {
	runItem
	Author    string
	StartTime string
	RawStatus string
}

type IconWithTooltip struct {
	Icon    string
	Tooltip string
	Color   string
}

func NewPRStatusPage(logger logr.Logger, auth auth.Authentication, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		allTests := tests.GetAllRunning()
		if len(allTests) == 0 {
			allTests = append(allTests, &demotest)
		}

		rawList := make([]runItem, len(allTests))
		for i, run := range allTests {
			rawList[i] = runItem{
				Organization: run.Event.GetOwnerName(),
				Repository:   run.Event.GetRepositoryName(),
				PR:           run.Event.ID,
				Testrun:      run.Testrun.GetName(),
				Phase:        PhaseIcon[util.TestrunStatusPhase(run.Testrun)],
				Progress:     util.TestrunProgress(run.Testrun),
				ArgoURL:      "",
			}
		}
		params := map[string]interface{}{
			"tests": rawList,
		}

		p.handleSimplePage("pr-status.html", params)(w, r)
	}
}

func NewPRStatusDetailPage(logger logr.Logger, auth auth.Authentication, basePath string) http.HandlerFunc {
	p := Page{log: logger, auth: auth, basePath: basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		trName := mux.Vars(r)["testrun"]
		if trName == "" {
			logger.Info("testrun is not defined")
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}
		allTests := tests.GetAllRunning()
		if len(allTests) == 0 {
			allTests = append(allTests, &demotest)
		}

		var run *tests.Run
		for _, r := range allTests {
			if r.Testrun.GetName() == trName {
				run = r
				break
			}
		}
		if run == nil {
			logger.Error(nil, "testrun cannot be found", "testrun", trName)
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		var (
			statusTable = &strings.Builder{}
			startTime   = ""
		)

		if len(run.Testrun.Status.Steps) != 0 {
			output.RenderStatusTable(statusTable, run.Testrun.Status.Steps)
		}
		if run.Testrun.Status.StartTime != nil {
			startTime = run.Testrun.Status.StartTime.Format(time.RFC822)
		}

		item := runDetailedItem{
			runItem: runItem{
				Organization: run.Event.GetOwnerName(),
				Repository:   run.Event.GetRepositoryName(),
				PR:           run.Event.ID,
				Testrun:      run.Testrun.GetName(),
				Phase:        PhaseIcon[util.TestrunStatusPhase(run.Testrun)],
				Progress:     util.TestrunProgress(run.Testrun),
			},
			Author:    run.Event.GetAuthorName(),
			StartTime: startTime,
			RawStatus: statusTable.String(),
		}

		p.handleSimplePage("pr-status-detailed.html", item)(w, r)
	}
}

var owner = "owner"
var repo = "repo"
var author = "demo-user"
var startTime = metav1.NewTime(time.Now())
var demotest = tests.Run{
	Testrun: &v1beta1.Testrun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-tr",
		},
		Status: v1beta1.TestrunStatus{
			StartTime: &startTime,
			Phase:     v1beta1.PhaseStatusRunning,
			Steps: []*v1beta1.StepStatus{
				{
					Phase: v1beta1.PhaseStatusRunning,
				},
			},
		},
	},
	Event: &github.GenericRequestEvent{
		ID:     3,
		Number: 0,
		Repository: &github2.Repository{
			Owner: &github2.User{
				Login: &owner,
			},
			Name: &repo,
		},
		Author: &github2.User{Login: &author},
	},
}
