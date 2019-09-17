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
//
package events

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
	"net/http"
)

func HandleWebhook(log logr.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func (w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		var gh_webhook github.PullRequestReviewCommentEvent
		if err := decoder.Decode(&gh_webhook); err != nil {
			log.Error(err, "unable to decode json payload")
			http.Error(w, "unable to read body", http.StatusInternalServerError)
			return
		}

		fmt.Println(util.PrettyPrintStruct(gh_webhook))
		w.Write([]byte{})
	}
}
