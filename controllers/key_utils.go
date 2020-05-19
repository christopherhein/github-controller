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

	"github.com/google/go-github/v28/github"
	"go.hein.dev/github-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

func (r *KeyReconciler) addFinalizer(ctx context.Context, key *v1alpha1.Key) error {
	key.ObjectMeta.Finalizers = append(key.ObjectMeta.Finalizers, keyFinalizerName)
	if err := r.Client.Update(ctx, key); err != nil {
		return err
	}

	return r.updateKeyStatus(ctx, key, v1alpha1.CreatingStatus)
}

func (r *KeyReconciler) handleDeletion(ctx context.Context, key *v1alpha1.Key) error {
	keyID := key.Status.GitHubKeyID
	repo := key.Status.GitHubRepository
	org := key.Status.GitHubOrganization

	if keyID != 0 && repo != "" && org != "" {
		_, resp, err := r.GitClient.GetKey(ctx, org, repo, keyID)
		if err != nil && !isNotFound(resp) {
			return err
		}

		if r.ActualDelete {
			r.Log.Info("actual delete true", "deleting", fmt.Sprintf("%s/%s/%s", org, repo, key.Name))
			if err := r.GitClient.DeleteKey(ctx, org, repo, keyID); err != nil {
				return err
			}
		}
	}

	key.ObjectMeta.Finalizers = removeString(key.ObjectMeta.Finalizers, keyFinalizerName)
	if err := r.Client.Update(context.Background(), key); err != nil {
		return err
	}
	return nil
}

func (r *KeyReconciler) updateKeyStatusCreatingPublicKey(ctx context.Context, nsn types.NamespacedName, publicKey string) error {
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var key v1alpha1.Key
		if err := r.Client.Get(ctx, nsn, &key); err != nil {
			return err
		}

		keyCopy := key.DeepCopy()
		keyCopy.Status = v1alpha1.KeyStatus{
			Status:    v1alpha1.CreatingStatus,
			PublicKey: publicKey,
		}

		return r.Client.Status().Update(ctx, keyCopy)
	}); err != nil {
		return err
	}
	return nil
}

func (r *KeyReconciler) updateKeyStatusDetails(ctx context.Context, repo *v1alpha1.Repository, ghKey *github.Key, key *v1alpha1.Key) error {
	nsn := types.NamespacedName{Namespace: key.Namespace, Name: key.Name}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var key v1alpha1.Key
		if err := r.Client.Get(ctx, nsn, &key); err != nil {
			return err
		}

		keyCopy := key.DeepCopy()
		keyCopy.Status.Status = v1alpha1.SyncedStatus
		keyCopy.Status.URL = fmt.Sprintf("https://github.com/%s/%s/settings/keys", repo.Spec.Organization, repo.Name)
		keyCopy.Status.GitHubKeyID = ghKey.GetID()
		keyCopy.Status.GitHubRepository = repo.GetName()
		keyCopy.Status.GitHubOrganization = repo.Spec.Organization

		return r.Client.Status().Update(ctx, keyCopy)
	}); err != nil {
		return err
	}
	return nil
}

func (r *KeyReconciler) updateKeyStatus(ctx context.Context, key *v1alpha1.Key, status v1alpha1.StatusReason) error {
	nsn := types.NamespacedName{Namespace: key.Namespace, Name: key.Name}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var key v1alpha1.Key
		if err := r.Client.Get(ctx, nsn, &key); err != nil {
			return err
		}

		if key.Status.Status == status {
			return nil // no need to update
		}

		keyCopy := key.DeepCopy()
		keyCopy.Status.Status = status

		return r.Client.Status().Update(ctx, keyCopy)
	}); err != nil {
		return err
	}
	return nil
}

// asOwner returns an OwnerReference set as the key CR
func asOwner(key *v1alpha1.Key) metav1.OwnerReference {
	isController := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         key.APIVersion,
		Kind:               key.Kind,
		Name:               key.Name,
		UID:                key.UID,
		Controller:         &isController,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}
