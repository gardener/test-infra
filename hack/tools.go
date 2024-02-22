//go:build tools
// +build tools

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// 
// SPDX-License-Identifier: Apache-2.0

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/gardener/gardener/.github"
	_ "github.com/gardener/gardener/.github/ISSUE_TEMPLATE"
	_ "github.com/gardener/gardener/hack"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "golang.org/x/lint/golint"

	_ "k8s.io/code-generator"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)
