// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package tests

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/gardener/test-infra/pkg/util/output"
	"strings"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
)

const (
	githubContext = "Test Machinery"
)

var GitHubState = map[argov1.NodePhase]github.State{
	tmv1beta1.PhaseStatusInit:    github.StatePending,
	tmv1beta1.PhaseStatusPending: github.StatePending,
	tmv1beta1.PhaseStatusRunning: github.StatePending,
	tmv1beta1.PhaseStatusSuccess: github.StateSuccess,
	tmv1beta1.PhaseStatusSkipped: github.StatePending,
	tmv1beta1.PhaseStatusFailed:  github.StateFailure,
	tmv1beta1.PhaseStatusError:   github.StateError,
	tmv1beta1.PhaseStatusTimeout: github.StateError,
}

type StatusUpdater struct {
	log    logr.Logger
	client github.Client
	event  *github.GenericRequestEvent

	commentID       int64
	lastState       github.State
	lastCommentHash []byte
}

func NewStatusUpdater(log logr.Logger, ghClient github.Client, event *github.GenericRequestEvent) *StatusUpdater {
	return &StatusUpdater{
		log:    log,
		client: ghClient,
		event:  event,
	}
}

func NewStatusUpdaterFromCommentID(log logr.Logger, ghClient github.Client, event *github.GenericRequestEvent, commentID int64) *StatusUpdater {
	return &StatusUpdater{
		log:       log,
		client:    ghClient,
		event:     event,
		commentID: commentID,
	}
}

func (u *StatusUpdater) Init(ctx context.Context, tr *tmv1beta1.Testrun) error {
	commentID, err := u.client.Comment(ctx, u.event, FormatInitStatus(tr))
	if err != nil {
		return err
	}
	u.commentID = commentID

	if err := u.client.UpdateStatus(ctx, u.event, github.StatePending, githubContext, tr.Name); err != nil {
		return err
	}

	return nil
}

func (u *StatusUpdater) GetCommentID() int64 {
	return u.commentID
}

// Update updates the comment and the github state of the current PR
func (u *StatusUpdater) Update(ctx context.Context, tr *tmv1beta1.Testrun, dashboardUrl string) error {
	comment := FormatStatus(tr, dashboardUrl)
	if err := u.UpdateComment(comment); err != nil {
		return err
	}

	state := GitHubState[util.TestrunStatusPhase(tr)]
	if err := u.UpdateStatus(ctx, state, tr.Name); err != nil {
		return err
	}

	return nil
}

// UpdateComment updates the current comment of the status updater
func (u *StatusUpdater) UpdateComment(comment string) error {
	if u.commentID == 0 {
		return nil
	}
	h := sha1.New()
	h.Write([]byte(comment))
	commentHash := h.Sum([]byte{})

	if bytes.Compare(commentHash, u.lastCommentHash) != 0 {
		if err := u.client.UpdateComment(u.event, u.commentID, comment); err != nil {
			return err
		}
		u.log.V(3).Info("updated status comment")
		u.lastCommentHash = commentHash
	}
	return nil
}

// UpdateStatus updates the GitHub status of the current PR
func (u *StatusUpdater) UpdateStatus(ctx context.Context, state github.State, description string) error {
	if state != u.lastState {
		if err := u.client.UpdateStatus(ctx, u.event, state, githubContext, description); err != nil {
			return err
		}
		u.log.V(3).Info("updated commit status")
		u.lastState = state
	}
	return nil
}

func FormatInitStatus(tr *tmv1beta1.Testrun) string {
	format := ":seedling: Successfully started test with id `%s`!"
	return fmt.Sprintf(format, tr.Name)
}

func FormatStatus(tr *tmv1beta1.Testrun, dashboardUrl string) string {
	var (
		testrunName = tr.Name
		statusTable = &strings.Builder{}
	)

	if len(tr.Status.Steps) != 0 {
		output.RenderStatusTable(statusTable, tr.Status.Steps)
	}
	if dashboardUrl != "" {
		testrunName = fmt.Sprintf("[%s](%s)", tr.Name, dashboardUrl)
	}

	format := `
Testrun: %s
Workflow: %s
Phase: %s
<pre>
%s
</pre>
`
	return fmt.Sprintf(format, testrunName, tr.Status.Workflow, util.TestrunStatusPhase(tr), statusTable.String())
}
