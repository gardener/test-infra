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
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	metadata2 "github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/tm-bot/ui/pages/pagination"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/gorilla/mux"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
	"time"
)

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

		argoHostURL, _ := testrunner.GetArgoHost(ctx, p.runs.GetClient())
		grafanaHostURL, _ := testrunner.GetGrafanaHost(ctx, p.runs.GetClient())
		metadata := metadata2.FromTestrun(tr)
		startTime := ""
		if tr.Status.StartTime != nil {
			startTime = tr.Status.StartTime.Format(time.RFC822)
		}
		d := time.Duration(tr.Status.Duration) * time.Second
		if tr.Status.Duration == 0 && !tr.Status.StartTime.IsZero() {
			d = time.Since(tr.Status.StartTime.Time)
			d = d / time.Second * time.Second // remove unnecessary milliseconds
		}

		statusTable := &strings.Builder{}
		if len(tr.Status.Steps) != 0 {
			output.RenderStatusTable(statusTable, tr.Status.Steps)
		}

		item := detailedTestrunItem{
			testrunItem: testrunItem{
				testrun:   tr,
				ID:        tr.GetName(),
				Namespace: tr.GetNamespace(),
				Phase:     PhaseIcon(tr.Status.Phase),
				StartTime: startTime,
				Duration:  d.String(),
				Progress:  util.TestrunProgress(tr),
				Dimension: metadata.GetDimensionFromMetadata("/"),
			},
			Steps:     make(testrunStepStatusItemList, len(tr.Status.Steps)),
			RawStatus: statusTable.String(),
		}
		if retries, ok := tr.Annotations[common.AnnotationRetries]; ok {
			item.Retries = retries
		}
		if prevAttempt, ok := tr.Annotations[common.AnnotationPreviousAttempt]; ok {
			item.PreviousAttempt = prevAttempt
		}
		if runID, ok := tr.Labels[common.LabelTestrunExecutionGroup]; ok {
			item.RunID = runID
		}
		if argoHostURL != "" {
			item.ArgoURL = testrunner.GetArgoURLFromHost(argoHostURL, tr)
		}
		if grafanaHostURL != "" {
			item.GrafanaURL = testrunner.GetGrafanaURLFromHostForWorkflow(grafanaHostURL, tr.Status.Workflow)
		}

		for i, step := range tr.Status.Steps {
			startTime := ""
			if step.StartTime != nil {
				startTime = step.StartTime.Format(time.RFC822)
			}
			d := time.Duration(step.Duration) * time.Second
			if step.Duration == 0 && !step.StartTime.IsZero() {
				d = time.Since(step.StartTime.Time)
				d = d / time.Second * time.Second // remove unnecessary milliseconds
			}
			item.Steps[i] = testrunStepStatusItem{
				step:      step,
				Name:      step.TestDefinition.Name,
				Step:      step.Position.Step,
				Phase:     PhaseIcon(step.Phase),
				StartTime: startTime,
				Duration:  d.String(),
				Location:  fmt.Sprintf("%s:%s", step.TestDefinition.Location.Repo, step.TestDefinition.Location.Revision),
				IsSystem:  util.IsSystemStep(step),
			}
			if grafanaHostURL != "" {
				item.Steps[i].GrafanaURL = testrunner.GetGrafanaURLFromHostForStep(grafanaHostURL, tr.Status.Workflow, step.TestDefinition.Name)
			}
		}

		sort.Sort(item.Steps)

		p.handleSimplePage("testrun.html", item)(w, r)
	}
}

type testrunItemList []testrunItem

func (l testrunItemList) GetPaginatedList(from, to int) pagination.Interface {
	to++
	if from > len(l) {
		return make(testrunItemList, 0)
	}
	if to >= len(l) {
		return l[from:]
	}
	return l[from:to]
}

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
