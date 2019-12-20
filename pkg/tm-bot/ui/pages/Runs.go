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
	"context"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
	"time"
)

type testrunItem struct {
	testrun   *v1beta1.Testrun
	ID        string
	Namespace string
	RunID     string
	Phase     IconWithTooltip
	StartTime string
	Duration  string
	Progress  string

	Dimension string

	ArgoURL    string
	GrafanaURL string
}

type detailedTestrunItem struct {
	testrunItem
	Steps     testrunStepStatusItemList
	RawStatus string
}

type testrunStepStatusItem struct {
	step      *v1beta1.StepStatus
	Name      string
	Step      string
	Phase     IconWithTooltip
	StartTime string
	Duration  string
	Location  string

	IsSystem bool

	GrafanaURL string
}

type rungroupItem struct {
	testruns  []*v1beta1.Testrun
	phase     argov1alpha1.NodePhase
	startTime *metav1.Time
	completed int

	DisplayName string
	Name        string
	StartTime   string
	State       string
	Phase       IconWithTooltip
}

func NewTestrunsPage(p *Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		defer ctx.Done()

		listOpts := client.MatchingLabels(map[string]string{})
		rg, rgOk := r.URL.Query()["runID"]
		if rgOk {
			listOpts = client.MatchingLabels(map[string]string{common.LabelTestrunRunID: rg[0]})
		}
		runs := &v1beta1.TestrunList{}
		if err := p.runs.GetClient().List(ctx, runs, listOpts); err != nil {
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		argoHostURL, _ := testrunner.GetArgoHost(p.runs.GetClient())
		grafanaHostURL, _ := testrunner.GetGrafanaHost(p.runs.GetClient())

		testrunsList := make(testrunItemList, len(runs.Items))
		runsList := make(rungroupItemList, 0)
		for i, tr := range runs.Items {
			testrun := tr

			if !rgOk {
				runsList.Add(&testrun)
			}
			metadata := testrunner.MetadataFromTestrun(&tr)
			startTime := ""
			if tr.Status.StartTime != nil {
				startTime = tr.Status.StartTime.Format(time.RFC822)
			}
			d := time.Duration(tr.Status.Duration) * time.Second
			testrunsList[i] = testrunItem{
				testrun:   &testrun,
				ID:        tr.GetName(),
				Namespace: tr.GetNamespace(),
				RunID:     tr.GetLabels()[common.LabelTestrunRunID],
				Phase:     PhaseIcon[util.TestrunStatusPhase(&tr)],
				StartTime: startTime,
				Duration:  d.String(),
				Progress:  util.TestrunProgress(&tr),
				Dimension: metadata.GetDimensionFromMetadata(),
			}
			if argoHostURL != "" {
				testrunsList[i].ArgoURL = testrunner.GetArgoURLFromHost(argoHostURL, &tr)
			}
			if grafanaHostURL != "" {
				testrunsList[i].GrafanaURL = testrunner.GetGrafanaURLFromHostForWorkflow(grafanaHostURL, &tr)
			}
		}

		sort.Sort(testrunsList)
		sort.Sort(runsList)
		params := map[string]interface{}{
			"rungroup":  rg,
			"tests":     testrunsList,
			"rungroups": runsList,
		}
		if len(testrunsList) > 50 {
			params["tests"] = testrunsList[:50] // todo add pagination to not cut at 50 items
		}
		if len(runsList) > 6 {
			params["rungroups"] = runsList[:6]
		}

		p.handleSimplePage("testruns.html", params)(w, r)
	}
}

func NewTestrunPage(p *Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		defer ctx.Done()

		trName := client.ObjectKey{
			Name:      mux.Vars(r)["testrun"],
			Namespace: mux.Vars(r)["namespace"],
		}

		tr := &v1beta1.Testrun{}
		if err := p.runs.GetClient().Get(ctx, trName, tr); err != nil {
			http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
			return
		}

		argoHostURL, _ := testrunner.GetArgoHost(p.runs.GetClient())
		grafanaHostURL, _ := testrunner.GetGrafanaHost(p.runs.GetClient())
		metadata := testrunner.MetadataFromTestrun(tr)
		startTime := ""
		if tr.Status.StartTime != nil {
			startTime = tr.Status.StartTime.Format(time.RFC822)
		}
		d := time.Duration(tr.Status.Duration) * time.Second

		statusTable := &strings.Builder{}
		if len(tr.Status.Steps) != 0 {
			output.RenderStatusTable(statusTable, tr.Status.Steps)
		}

		item := detailedTestrunItem{
			testrunItem: testrunItem{
				testrun:   tr,
				ID:        tr.GetName(),
				Namespace: tr.GetNamespace(),
				RunID:     tr.GetLabels()[common.LabelTestrunRunID],
				Phase:     PhaseIcon[util.TestrunStatusPhase(tr)],
				StartTime: startTime,
				Duration:  d.String(),
				Progress:  util.TestrunProgress(tr),
				Dimension: metadata.GetDimensionFromMetadata(),
			},
			Steps:     make(testrunStepStatusItemList, len(tr.Status.Steps)),
			RawStatus: statusTable.String(),
		}
		if argoHostURL != "" {
			item.ArgoURL = testrunner.GetArgoURLFromHost(argoHostURL, tr)
		}
		if grafanaHostURL != "" {
			item.GrafanaURL = testrunner.GetGrafanaURLFromHostForWorkflow(grafanaHostURL, tr)
		}

		for i, step := range tr.Status.Steps {
			startTime := ""
			if step.StartTime != nil {
				startTime = step.StartTime.Format(time.RFC822)
			}
			d := time.Duration(step.Duration) * time.Second
			item.Steps[i] = testrunStepStatusItem{
				step:      step,
				Name:      step.TestDefinition.Name,
				Step:      step.Position.Step,
				Phase:     PhaseIcon[step.Phase],
				StartTime: startTime,
				Duration:  d.String(),
				Location:  fmt.Sprintf("%s:%s", step.TestDefinition.Location.Repo, step.TestDefinition.Location.Revision),
				IsSystem:  util.IsSystemStep(step),
			}
			if grafanaHostURL != "" {
				item.Steps[i].GrafanaURL = testrunner.GetGrafanaURLFromHostForStep(grafanaHostURL, tr, step)
			}
		}

		sort.Sort(item.Steps)

		p.handleSimplePage("testrun.html", item)(w, r)
	}
}

type testrunItemList []testrunItem

func (l testrunItemList) Len() int      { return len(l) }
func (l testrunItemList) Swap(a, b int) { l[a], l[b] = l[b], l[a] }
func (l testrunItemList) Less(a, b int) bool {
	aTr := l[a].testrun
	bTr := l[b].testrun
	if aTr.Status.Phase != bTr.Status.Phase {
		if aTr.Status.Phase == v1beta1.PhaseStatusRunning {
			return true
		}
		if bTr.Status.Phase == v1beta1.PhaseStatusRunning {
			return false
		}
	}
	if aTr.Status.StartTime == nil || bTr.Status.StartTime == nil {
		return true
	}

	return bTr.Status.StartTime.Before(aTr.Status.StartTime)
}

type testrunStepStatusItemList []testrunStepStatusItem

func (l testrunStepStatusItemList) Len() int      { return len(l) }
func (l testrunStepStatusItemList) Swap(a, b int) { l[a], l[b] = l[b], l[a] }
func (l testrunStepStatusItemList) Less(a, b int) bool {
	if l[a].step.Phase != l[b].step.Phase {
		if l[a].step.Phase == v1beta1.PhaseStatusRunning {
			return true
		}
		if l[b].step.Phase == v1beta1.PhaseStatusRunning {
			return false
		}

		if l[a].step.Phase == v1beta1.PhaseStatusInit {
			return false
		}
		if l[b].step.Phase == v1beta1.PhaseStatusInit {
			return true
		}
	}
	if l[a].step.StartTime == nil || l[b].step.StartTime == nil {
		return true
	}
	return l[a].step.StartTime.Before(l[b].step.StartTime)
}

type rungroupItemList []rungroupItem

func (l rungroupItemList) Len() int      { return len(l) }
func (l rungroupItemList) Swap(a, b int) { l[a], l[b] = l[b], l[a] }
func (l *rungroupItemList) Add(tr *v1beta1.Testrun) {
	list := *l
	runId, ok := tr.GetLabels()[common.LabelTestrunRunID]
	if !ok {
		return
	}

	isCompleted := 0
	if util.Completed(tr.Status.Phase) {
		isCompleted = 1
	}

	for i, run := range list {
		if run.Name == runId {
			list[i].testruns = append(run.testruns, tr)
			list[i].phase = mergePhases(run.phase, tr.Status.Phase)
			list[i].Phase = PhaseIcon[list[i].phase]
			list[i].completed = list[i].completed + isCompleted
			list[i].State = fmt.Sprintf("%d/%d Testruns are completed", list[i].completed, len(list[i].testruns))
			return
		}
	}

	*l = append(*l, rungroupItem{
		testruns:    []*v1beta1.Testrun{tr},
		phase:       util.TestrunStatusPhase(tr),
		startTime:   tr.Status.StartTime,
		completed:   1,
		DisplayName: testgroupDisplayName(tr),
		Name:        runId,
		StartTime:   tr.Status.StartTime.Format(time.RFC822),
		State:       fmt.Sprintf("%d/%d Testruns are completed", isCompleted, 1),
		Phase:       PhaseIcon[util.TestrunStatusPhase(tr)],
	})
}
func (l rungroupItemList) Less(a, b int) bool {
	if l[a].phase != l[b].phase {
		if l[a].phase == v1beta1.PhaseStatusRunning {
			return true
		}
		if l[b].phase == v1beta1.PhaseStatusRunning {
			return false
		}

		if l[a].phase == v1beta1.PhaseStatusInit {
			return false
		}
		if l[b].phase == v1beta1.PhaseStatusInit {
			return true
		}
	}
	if l[a].startTime == nil || l[b].startTime == nil {
		return true
	}
	return l[b].startTime.Before(l[a].startTime)
}

func mergePhases(a, b argov1alpha1.NodePhase) argov1alpha1.NodePhase {
	if a == v1beta1.PhaseStatusRunning || b == v1beta1.PhaseStatusRunning {
		return v1beta1.PhaseStatusRunning
	}
	if a == v1beta1.PhaseStatusFailed || b == v1beta1.PhaseStatusFailed {
		return v1beta1.PhaseStatusFailed
	}
	if a == v1beta1.PhaseStatusError || b == v1beta1.PhaseStatusError {
		return v1beta1.PhaseStatusError
	}
	if a == v1beta1.PhaseStatusTimeout || b == v1beta1.PhaseStatusTimeout {
		return v1beta1.PhaseStatusTimeout
	}
	return a
}

func testgroupDisplayName(tr *v1beta1.Testrun) string {
	landscape, ok := tr.GetAnnotations()[common.AnnotationLandscape]
	if ok {
		return landscape
	}
	return "Unknown"
}
