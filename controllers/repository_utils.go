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

package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

func (r *RepositoryReconciler) addFinalizer(ctx context.Context, repository *v1alpha1.Repository) error {
	repository.ObjectMeta.Finalizers = append(repository.ObjectMeta.Finalizers, repoFinalizerName)
	if err := r.Client.Update(ctx, repository); err != nil {
		return err
	}

	return r.updateStatus(ctx, repository, v1alpha1.CreatingStatus)
}

func (r *RepositoryReconciler) handleDeletion(ctx context.Context, repository *v1alpha1.Repository) error {
	_, resp, err := r.GitClient.GetRepo(ctx, repository.Spec.Organization, repository.Name)
	if err != nil && !isNotFound(resp) {
		return err
	}

	if err := r.GitClient.DeleteRepo(ctx, repository.Spec.Organization, repository.Name); err != nil {
		return err
	}

	repository.ObjectMeta.Finalizers = removeString(repository.ObjectMeta.Finalizers, repoFinalizerName)
	if err := r.Client.Update(context.Background(), repository); err != nil {
		return err
	}
	return nil
}

func (r *RepositoryReconciler) updateRepositoryStatus(ctx context.Context, ghrepo *github.Repository, repository *v1alpha1.Repository) error {
	nsn := types.NamespacedName{Namespace: repository.Namespace, Name: repository.Name}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var repo v1alpha1.Repository
		if err := r.Client.Get(ctx, nsn, &repo); err != nil {
			return err
		}

		repoCopy := repo.DeepCopy()
		repoCopy.Status = v1alpha1.RepositoryStatus{
			Status:          v1alpha1.SyncedStatus,
			URL:             fmt.Sprintf("https://github.com/%s/%s", repository.Spec.Organization, repository.Name),
			ForkCount:       ghrepo.GetForksCount(),
			StargazersCount: ghrepo.GetStargazersCount(),
			WatchersCount:   ghrepo.GetWatchersCount(),
		}

		return r.Client.Status().Update(ctx, repoCopy)
	}); err != nil {
		return err
	}
	return nil
}

func (r *RepositoryReconciler) updateStatus(ctx context.Context, repository *v1alpha1.Repository, status v1alpha1.StatusReason) error {
	nsn := types.NamespacedName{Namespace: repository.Namespace, Name: repository.Name}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var repo v1alpha1.Repository
		if err := r.Client.Get(ctx, nsn, &repo); err != nil {
			return err
		}

		repoCopy := repo.DeepCopy()
		repoCopy.Status = v1alpha1.RepositoryStatus{
			Status: status,
			URL:    fmt.Sprintf("https://github.com/%s/%s", repository.Spec.Organization, repository.Name),
		}

		return r.Client.Status().Update(ctx, repoCopy)
	}); err != nil {
		return err
	}
	return nil
}

func isNotFound(r *github.Response) bool {
	if r.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
