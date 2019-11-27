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

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
)

// TestClient will generate a test stub
func TestClient() Client {
	return &testclient{
		created: false,
		deleted: false,
	}
}

type testclient struct {
	deleted bool
	created bool
}

// GetRepo will reachout to Github and create the repo
func (in *testclient) GetRepo(ctx context.Context, org, name string) (*github.Repository, *github.Response, error) {
	if in.created {
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

// CreateRepo will reachout to Github and create the repo
func (in *testclient) CreateRepo(ctx context.Context, org string, repo *v1alpha1.Repository) error {
	in.created = true
	in.deleted = false
	return nil
}

func (in *testclient) DeleteRepo(ctx context.Context, org, name string) error {
	in.created = true
	in.deleted = true
	return nil
}
