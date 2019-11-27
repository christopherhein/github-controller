/*
Copyright 2019 Christopher Hein <me@chrishein.com>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package git contains the git functions
package git

import (
	"context"
	"log"
	"net/http"

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
	"golang.org/x/oauth2"
)

// Client defines the interface to use with the controllers
type Client interface {
	// GetRepo will find the remote or error
	GetRepo(context.Context, string, string) (*github.Repository, *github.Response, error)

	// CreateRepo will create a repo based on the params
	CreateRepo(context.Context, string, *v1alpha1.Repository) error

	// DeleteRepo will delete the repo
	DeleteRepo(context.Context, string, string) error
}

type client struct {
	ts oauth2.TokenSource
	tc *http.Client
	c  *github.Client
}

// New creates a new git client
func New(ctx context.Context, token string) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return &client{
		ts: ts,
		tc: tc,
		c:  github.NewClient(tc),
	}
}

// GetRepo will reachout to Github and create the repo
func (in *client) GetRepo(ctx context.Context, org, name string) (repo *github.Repository, resp *github.Response, err error) {
	repo, resp, err = in.c.Repositories.Get(ctx, org, name)
	if err != nil {
		return repo, resp, err
	}
	return repo, resp, nil
}

// CreateRepo will reachout to Github and create the repo
func (in *client) CreateRepo(ctx context.Context, org string, repo *v1alpha1.Repository) error {
	r := newRepository(repo)
	_, _, err := in.c.Repositories.Create(ctx, org, r)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (in *client) DeleteRepo(ctx context.Context, org, name string) error {
	_, err := in.c.Repositories.Delete(ctx, org, name)
	if err != nil {
		return err
	}
	return nil
}

func newRepository(repo *v1alpha1.Repository) *github.Repository {
	return &github.Repository{
		Name:        &repo.Name,
		Description: &repo.Spec.Description,
		Homepage:    &repo.Spec.Homepage,
		Private:     &repo.Spec.Settings.Private,
		HasIssues:   &repo.Spec.Settings.Issues,
		HasWiki:     &repo.Spec.Settings.Wiki,
		HasProjects: &repo.Spec.Settings.Projects,
		IsTemplate:  &repo.Spec.Settings.Template,
	}
}
