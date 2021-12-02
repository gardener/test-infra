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

package github

import "context"

// IsAuthorized checks if the author of the event is authorized to perform actions on the service
func (c *client) IsAuthorized(authorizationType AuthorizationType, event *GenericRequestEvent) bool {
	if UserType(*event.Author.Type) == UserTypeBot {
		return false
	}
	ctx := context.Background()
	defer ctx.Done()

	switch authorizationType {
	case AuthorizationAll:
		return true
	case AuthorizationOrg:
		return c.isInOrganization(ctx, event)
	case AuthorizationTeam:
		return c.isInDefaultTeam(ctx, event)
	case AuthorizationCodeOwners:
		// todo: update to really parse the codeowners file with fallback to default team or org
		return c.isInRequestedTeam(ctx, event)
	case AuthorizationOrgAdmin:
		return c.isOrgAdmin(ctx, event)
	}
	return false
}

// isOrgAdmin checks if the author is organization admin
func (c *client) isOrgAdmin(ctx context.Context, event *GenericRequestEvent) bool {
	membership, _, err := c.client.Organizations.GetOrgMembership(ctx, event.GetAuthorName(), event.GetOwnerName())
	if err != nil {
		c.log.V(3).Info(err.Error())
		return false
	}
	if MembershipStatus(membership.GetState()) != MembershipStatusActive {
		return false
	}
	if MembershipRole(membership.GetRole()) == MembershipRoleAdmin {
		return true
	}
	return false
}

// isInOrganization checks if the author is in the organization
func (c *client) isInOrganization(ctx context.Context, event *GenericRequestEvent) bool {
	membership, _, err := c.client.Organizations.GetOrgMembership(ctx, event.GetAuthorName(), event.GetOwnerName())
	if err != nil {
		c.log.V(3).Info(err.Error())
		return false
	}
	if MembershipStatus(membership.GetState()) == MembershipStatusActive {
		return true
	}
	return false
}

// isInRequestedTeam checks if the author is in the requested PR team
func (c *client) isInRequestedTeam(ctx context.Context, event *GenericRequestEvent) bool {
	pr, err := c.GetPullRequest(ctx, event)
	if err != nil {
		return false
	}

	// use default team if there is no requested team
	if c.defaultTeam != nil && len(pr.RequestedTeams) == 0 {
		membership, _, err := c.client.Teams.GetTeamMembershipByID(ctx, c.defaultTeam.Organization.GetID(), c.defaultTeam.GetID(), event.GetAuthorName())
		if err != nil {
			c.log.V(3).Info(err.Error(), "team", c.defaultTeam.GetName())
			return false
		}
		if MembershipStatus(membership.GetState()) != MembershipStatusActive {
			return true
		}
		return false
	}

	for _, team := range pr.RequestedTeams {
		membership, _, err := c.client.Teams.GetTeamMembershipByID(ctx, team.Organization.GetID(), team.GetID(), event.GetAuthorName())
		if err != nil {
			c.log.V(3).Info(err.Error(), "team", team.GetName())
			return false
		}
		if MembershipStatus(membership.GetState()) == MembershipStatusActive {
			return true
		}
	}
	return false
}

// isInRequestedTeam checks if the author is in the requested PR team
func (c *client) isInDefaultTeam(ctx context.Context, event *GenericRequestEvent) bool {
	if c.defaultTeam == nil {
		c.log.Info("no default team defined", "repository", event.GetRepositoryName(), "owner", event.GetOwnerName())
		return false
	}
	membership, _, err := c.client.Teams.GetTeamMembershipByID(ctx, c.defaultTeam.Organization.GetID(), c.defaultTeam.GetID(), event.GetAuthorName())
	if err != nil {
		c.log.V(3).Info(err.Error(), "team", c.defaultTeam.GetName())
		return false
	}
	if MembershipStatus(membership.GetState()) == MembershipStatusActive {
		return true
	}
	return false
}
