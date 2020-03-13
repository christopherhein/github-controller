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
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "go.hein.dev/github-controller/api/v1alpha1"
	"go.hein.dev/github-controller/git"
)

var (
	repoFinalizerName = "repository.finalizers.github.go.hein.dev"
)

// RepositoryReconciler reconciles a Repository object
type RepositoryReconciler struct {
	Client       client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	GitClient    git.Client
	ActualDelete bool
}

// +kubebuilder:rbac:groups=github.go.hein.dev,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=github.go.hein.dev,resources=repositories/status,verbs=get;update;patch

// Reconcile is responsible for reconciling the request
func (r *RepositoryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var requeueafter time.Duration = 2 * time.Second
	ctx := context.Background()
	log := r.Log.WithValues("repository", req.NamespacedName)

	var repository v1alpha1.Repository
	if err := r.Client.Get(ctx, req.NamespacedName, &repository); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	organizationRepo := fmt.Sprintf("%s/%s", repository.Spec.Organization, repository.Name)

	log.Info("found local repository", "name", repository.Name)

	if repository.ObjectMeta.DeletionTimestamp.IsZero() &&
		!containsString(repository.GetFinalizers(), repoFinalizerName) {
		log.Info("adding finalizer", "name", repository.Name)
		if err := r.addFinalizer(ctx, &repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !repository.ObjectMeta.DeletionTimestamp.IsZero() &&
		containsString(repository.GetFinalizers(), repoFinalizerName) {
		log.Info("handle deletion", "name", repository.Name)
		if err := r.handleDeletion(ctx, &repository); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	repo, resp, err := r.GitClient.GetRepo(ctx, repository.Spec.Organization, repository.Name)

	if err != nil && isNotFound(resp) {
		log.Info("respository not found", "creating", organizationRepo)
		if err := r.GitClient.CreateRepo(ctx, repository.Spec.Organization, &repository); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueafter}, nil
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("found remote repository", "name", organizationRepo)

	// TODO: (christopherhein) Update Repo Checks

	if err := r.updateRepositoryStatusDetails(ctx, repo, &repository); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("updated local repo repository")

	return ctrl.Result{}, nil
}

// SetupWithManager configures the controller
func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Repository{}).
		Complete(r)
}
