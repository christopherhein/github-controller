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
	"net/http"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
)

// TestClient will generate a test stub
func TestClient() Client {
	return &testclient{
		RepositoryCreated: false,
		RepositoryDeleted: false,
	}
}

type testclient struct {
	IdCounter         int64
	RepositoryCreated bool
	RepositoryDeleted bool
	KeyCreated        bool
	KeyDeleted        bool
}

func (in *testclient) GetRepo(ctx context.Context, org, name string) (*github.Repository, *github.Response, error) {
	if in.RepositoryCreated {
		resp := &github.Response{
			Response: &http.Response{StatusCode: http.StatusOK},
		}
		return &github.Repository{}, resp, nil
	}
	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	return &github.Repository{}, resp, fmt.Errorf("not found")
}

func (in *testclient) CreateRepo(ctx context.Context, org string, repo *v1alpha1.Repository) error {
	in.RepositoryCreated = true
	in.RepositoryDeleted = false
	return nil
}

func (in *testclient) DeleteRepo(ctx context.Context, org, name string) error {
	in.RepositoryCreated = true
	in.RepositoryDeleted = true
	return nil
}

func (in *testclient) GetKey(ctx context.Context, org, repoName string, keyID int64) (*github.Key, *github.Response, error) {
	if in.KeyCreated {
		resp := &github.Response{
			Response: &http.Response{StatusCode: http.StatusOK},
		}
		return &github.Key{}, resp, nil
	}
	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	return &github.Key{}, resp, fmt.Errorf("not found")
}

func (in *testclient) CreateKey(ctx context.Context, org, repoName string, key *v1alpha1.Key, _ *corev1.Secret) (*github.Key, error) {
	in.KeyCreated = true
	in.KeyDeleted = false

	in.IdCounter++

	id := in.IdCounter
	title := "test-key"
	keyStr := "ssh-rsa test1234"
	readOnly := true
	return &github.Key{
			ID:       &id,
			Title:    &title,
			Key:      &keyStr,
			ReadOnly: &readOnly,
		},
		nil
}

func (in *testclient) DeleteKey(ctx context.Context, org, name string, keyID int64) error {
	in.KeyCreated = true
	in.KeyDeleted = true
	return nil
}
