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
	"fmt"
	"log"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
	"golang.org/x/oauth2"
)

// Client defines the interface to use with the controllers
type Client interface {
	// GetRepo will find the remote repo or error
	GetRepo(context.Context, string, string) (*github.Repository, *github.Response, error)

	// CreateRepo will create a repo based on the params
	CreateRepo(context.Context, string, *v1alpha1.Repository) error

	// DeleteRepo will delete the repo
	DeleteRepo(context.Context, string, string) error

	// GetKey will find the remote key or error
	GetKey(context.Context, string, string, int64) (*github.Key, *github.Response, error)

	// CreateKey will create a key in the repo based on the params
	CreateKey(context.Context, string, string, *v1alpha1.Key, *corev1.Secret) (*github.Key, error)

	// DeleteKey will delete the key from the repo
	DeleteKey(context.Context, string, string, int64) error
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

func (in *client) GetRepo(ctx context.Context, org, name string) (repo *github.Repository, resp *github.Response, err error) {
	repo, resp, err = in.c.Repositories.Get(ctx, org, name)
	if err != nil {
		return repo, resp, err
	}
	return repo, resp, nil
}

func (in *client) CreateRepo(ctx context.Context, org string, repo *v1alpha1.Repository) error {
	authUser, _, err := in.c.Users.Get(ctx, "")
	if err != nil {
		log.Fatal(err)
	}

	if authUser.GetLogin() == org {
		// username is equal to target organization
		org = "" // pass an empty string, only for repository creation
	}

	r := newRepository(repo)
	_, _, err = in.c.Repositories.Create(ctx, org, r)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (in *client) DeleteRepo(ctx context.Context, org, name string) error {
	resp, err := in.c.Repositories.Delete(ctx, org, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("WARNING\t%v", err)
		} else {
			return err
		}
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

func (in *client) GetKey(ctx context.Context, org, repoName string, keyID int64) (key *github.Key, resp *github.Response, err error) {
	key, resp, err = in.c.Repositories.GetKey(ctx, org, repoName, keyID)
	if err != nil {
		return key, resp, err
	}
	return key, resp, nil
}

func (in *client) CreateKey(ctx context.Context, org, repoName string, key *v1alpha1.Key, secret *corev1.Secret) (*github.Key, error) {
	k, err := newKey(key, secret)
	if err != nil {
		return k, err
	}

	ghKey, _, err := in.c.Repositories.CreateKey(ctx, org, repoName, k)
	if err != nil {
		log.Fatal(err)
	}

	return ghKey, nil
}

func (in *client) DeleteKey(ctx context.Context, org, name string, keyID int64) error {
	resp, err := in.c.Repositories.DeleteKey(ctx, org, name, keyID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("WARNING\t%v", err)
		} else {
			return err
		}
	}
	return nil
}

func newKey(key *v1alpha1.Key, secret *corev1.Secret) (*github.Key, error) {
	publicKey := string(secret.Data["identity.pub"])
	if publicKey == "" {
		return nil, fmt.Errorf("secret data key %q did not contain a public key", "identity.pub")
	}

	return &github.Key{
		Title:    &key.Name,
		Key:      &publicKey,
		ReadOnly: &key.Spec.ReadOnly,
	}, nil
}
