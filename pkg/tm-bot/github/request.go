// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package github

func (e *GenericRequestEvent) GetRepositoryKey() RepositoryKey {
	return RepositoryKey{Owner: e.GetOwnerName(), Repository: e.GetRepositoryName()}
}

func (e *GenericRequestEvent) GetRepositoryName() string {
	return e.Repository.GetName()
}

func (e *GenericRequestEvent) GetOwnerName() string {
	return e.Repository.GetOwner().GetLogin()
}

func (e *GenericRequestEvent) GetMessage() string {
	return e.Body
}

func (e *GenericRequestEvent) GetAuthorName() string {
	return e.Author.GetLogin()
}
