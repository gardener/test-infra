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

package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

// MaxMessage is the maximium size of a slack message according to slacks docs
// https://api.slack.com/changelog/2018-04-truncating-really-long-messages#:~:text=The%20text%20field%20of%20messages,but%20left%20the%20consequences%20ambiguous.
const MaxMessageLimit = 4000

// Client defines the interface to interact with the slack API
type Client interface {
	// PostMessage will sends a message as the token user to the specified channel
	PostMessage(channel string, message string) error

	// PostRawMessage will sends a raw message as the token user to the specified channel
	PostRawMessage(message MessageRequest) error
}

type slack struct {
	log   logr.Logger
	token string
}

// New creates a new slack client to interact with the slack API
func New(log logr.Logger, token string) (Client, error) {
	if len(token) == 0 {
		return nil, errors.New("token has to be defined")
	}
	return &slack{
		log:   log,
		token: token,
	}, nil
}

func (s *slack) PostMessage(channel string, message string) error {
	rawMessage := MessageRequest{
		Channel:     channel,
		AsUser:      true,
		Text:        message,
		UnfurlLinks: false,
		UnfurlMedia: false,
	}
	return s.PostRawMessage(rawMessage)
}

func (s *slack) PostRawMessage(message MessageRequest) error {
	slackReq, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewBuffer(slackReq))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.log.Error(err, "unable to close request body")
		}
	}()

	if resp.StatusCode >= 300 {
		return errors.New("unable to send slack message")
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	slackResp := &Response{}
	if err := json.Unmarshal(rawBody, &slackResp); err != nil {
		return err
	}

	if !slackResp.Ok {
		return errors.Wrap(errors.New(*slackResp.Error), "unable to send response")
	}

	return nil
}

// MessageRequest defines a default slack request for a message
type MessageRequest struct {
	Channel     string `json:"channel"`
	Text        string `json:"text,omitempty"`
	AsUser      bool   `json:"as_user,omitempty"`
	UnfurlLinks bool   `json:"unfurl_links"`
	UnfurlMedia bool   `json:"unfurl_media"`
}

// Response defines a slack response
type Response struct {
	Ok      bool         `json:"ok"`
	Message *interface{} `json:"message"`
	Error   *string      `json:"error"`
}
