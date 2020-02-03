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
	"github.com/gardener/test-infra/pkg/testrunner"
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

type IconWithTooltip struct {
	Icon    string
	Tooltip string
	Color   string
}

var PhaseIcon = func(phase v1alpha1.NodePhase) IconWithTooltip {
	switch phase {
	case v1beta1.PhaseStatusInit:
		return IconWithTooltip{
			Icon:    "schedule",
			Tooltip: fmt.Sprintf("%s phase: Testrun is waiting to be scheduled", v1beta1.PhaseStatusInit),
			Color:   "grey",
		}
	case v1beta1.PhaseStatusSkipped:
		return IconWithTooltip{
			Icon:    "remove",
			Tooltip: fmt.Sprintf("%s phase: Testrun was skipped", v1beta1.PhaseStatusSkipped),
			Color:   "grey",
		}
	case v1beta1.PhaseStatusPending:
		return IconWithTooltip{
			Icon:    "schedule",
			Tooltip: fmt.Sprintf("%s phase: Testrun is pending", v1beta1.PhaseStatusSkipped),
			Color:   "orange",
		}
	case v1beta1.PhaseStatusRunning:
		return IconWithTooltip{
			Icon:    "autorenew",
			Tooltip: fmt.Sprintf("%s phase: Testrun is running", v1beta1.PhaseStatusRunning),
			Color:   "orange",
		}
	case v1beta1.PhaseStatusSuccess:
		return IconWithTooltip{
			Icon:    "done",
			Tooltip: fmt.Sprintf("%s phase: Testrun succeeded", v1beta1.PhaseStatusSuccess),
			Color:   "green",
		}
	case v1beta1.PhaseStatusFailed:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun failed", v1beta1.PhaseStatusFailed),
			Color:   "red",
		}
	case v1beta1.PhaseStatusError:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun errored", v1beta1.PhaseStatusError),
			Color:   "red",
		}
	case v1beta1.PhaseStatusTimeout:
		return IconWithTooltip{
			Icon:    "clear",
			Tooltip: fmt.Sprintf("%s phase: Testrun run longer than the specified timeout", v1beta1.PhaseStatusTimeout),
			Color:   "red",
		}
	default:
		return IconWithTooltip{
			Icon:    "info",
			Tooltip: fmt.Sprintf("%s phase", phase),
			Color:   "grey",
		}

	}
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

func NewPRStatusPage(p *Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAuthenticated := true
		_, err := p.auth.GetAuthContext(r)
		if err != nil {
			p.log.V(3).Info(err.Error())
			isAuthenticated = false
		}

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
				Phase:        PhaseIcon(util.TestrunStatusPhase(run.Testrun)),
				Progress:     util.TestrunProgress(run.Testrun),
			}
			if isAuthenticated {
				rawList[i].ArgoURL, _ = testrunner.GetArgoURL(p.runs.GetClient(), run.Testrun)
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
				Phase:        PhaseIcon(util.TestrunStatusPhase(run.Testrun)),
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
			Workflow: "tm-test49l44",
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
