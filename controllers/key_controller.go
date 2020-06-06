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
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v28/github"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "go.hein.dev/github-controller/api/v1alpha1"
	"go.hein.dev/github-controller/git"
	"go.hein.dev/github-controller/keygen"
)

var (
	keyFinalizerName = "key.finalizers.github.go.hein.dev"
)

// KeyReconciler reconciles a Key object
type KeyReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	GitClient    git.Client
	ActualDelete bool
}

// +kubebuilder:rbac:groups=github.go.hein.dev,resources=keys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=github.go.hein.dev,resources=keys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create

func (r *KeyReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var requeueAfter = 2 * time.Second
	ctx := context.Background()
	log := r.Log.WithValues("key", req.NamespacedName)

	var key v1alpha1.Key
	if err := r.Client.Get(ctx, req.NamespacedName, &key); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// handle finalizers before any other reconcile logic can fail
	if !key.ObjectMeta.DeletionTimestamp.IsZero() &&
		containsString(key.GetFinalizers(), keyFinalizerName) {
		log.Info("handle deletion", "name", key.Name)
		if err := r.handleDeletion(ctx, &key); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// fetch or create the desired Secret
	// the child secret defaults to the same namespace/name as the key
	secretRef := req.NamespacedName
	// ensure the desired Secret is referenced by the desired namespace/name
	if key.Spec.SecretTemplate.NameOverride != "" {
		secretRef.Name = key.Spec.SecretTemplate.NameOverride
	}
	if key.Spec.SecretTemplate.TargetNamespace != "" {
		secretRef.Namespace = key.Spec.SecretTemplate.TargetNamespace
	}
	log = log.WithValues("secret", secretRef)
	var secret corev1.Secret
	if err := r.Client.Get(ctx, secretRef, &secret); err != nil {
		if errors.IsNotFound(err) {
			// any previous secret no longer exists -- optimistically delete the old key
			if key.Status.GitHubKeyID != 0 {
				r.updateKeyStatus(ctx, &key, v1alpha1.DeletingStatus)
				err = r.GitClient.DeleteKey(ctx, key.Status.GitHubOrganization, key.Status.GitHubRepository, key.Status.GitHubKeyID)
				if err != nil {
					return ctrl.Result{RequeueAfter: requeueAfter}, err
				}
			}

			r.updateKeyStatus(ctx, &key, v1alpha1.CreatingStatus)
			log.Info("referenced secret does not exist, generating new key")
			privateKey, privKeyErr := keygen.GenerateRSAPrivateKey(4096)
			if privKeyErr != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, privKeyErr
			}
			publicKey, publicKeyErr := keygen.GenerateRSAPublicKey(&privateKey.PublicKey)
			if publicKeyErr != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, publicKeyErr
			}
			privateKeyPem := keygen.EncodeRSAPrivateKeyToPEM(privateKey)

			// Update the Key.status first
			// When Secret creation fails, Reconcile will overwrite the old publicKey with a new one
			// (If we created the Secret first but failed to update the status, it would orphan the Secret and error out the Key on next Reconcile.)
			// note:
			// 	This could cause a race condition if a client attempts to imperatively use the first value that is set as the PublicKey as it may be overwritten.
			//  Clients should also check the Status value and potentially wait until the key is copied to GitHub.
			// 	Reconcile will indefinitely update the PublicKey status with a new value while Secret creation is unauthorized.
			if err := r.updateKeyStatusCreatingPublicKey(ctx, req.NamespacedName, string(publicKey)); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, err
			}

			secret.SetName(secretRef.Name)
			secret.SetNamespace(secretRef.Namespace)
			secret.ObjectMeta.SetLabels(key.Spec.SecretTemplate.DeepCopy().Labels)
			secret.ObjectMeta.SetAnnotations(key.Spec.SecretTemplate.DeepCopy().Annotations)
			secret.Data = map[string][]byte{
				"identity":     privateKeyPem,
				"identity.pub": publicKey,
			}
			// TODO(stealthybox): v1.18 clients can make this Secret immutable -- should we?
			// 	It might be useful but very weird to allow other clients to add keys to this Secret.

			if secretRef.Namespace == req.Namespace {
				// Owner refs only work in the same Namespace: https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/
				// Queuing the owning Key based on the Secret changing or being deleted only works within a Namespace.
				secret.SetOwnerReferences([]metav1.OwnerReference{asOwner(&key)})
			}

			if err := r.Client.Create(ctx, &secret); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, err
			}
			log.Info("created new secret")

			// Block until the cache updates, so that our Secret returns without failing the Get() on the next Reconcile
			// If we don't do this, it breaks the publicKey matching logic between Key and the child Secret
			// TODO(stealthybox): Can we remove this by using a non-cached client for Secrets CRUD?  Why does Create() not update the cache?
			attempts := 1
			for ; attempts <= 10; attempts++ {
				time.Sleep(10 * time.Millisecond) // our cache is slightly sleepy -- it needs an espresso
				err = r.Client.Get(ctx, secretRef, &secret)
				if err == nil {
					log.Info("succeeded fetching secret post-creation", "fetch-attempts", attempts)
					break
				}
				// TODO(stealthybox): This is where we would need to register a Watch or set-member for Secrets in non matching targetNamespaces.
				// 	If related Secrets outside of the Namespace are deleted, it should queue this related Key.
				// 	For Secrets in particular, this could also be implemented by filtering the whole collection for Key related Owner refs or using a Finalizer.
				// 	A re-sync masks this issue.
			}
			if err != nil {
				log.Error(nil, "failed to fetch secret post-creation", "fetch-attempts", attempts)
				return ctrl.Result{RequeueAfter: requeueAfter}, err
			}

			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
		log.Error(err, "unexpected error fetching referenced secret")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	if secret.Data == nil || strings.TrimSpace(key.Status.PublicKey) != strings.TrimSpace(string(secret.Data["identity.pub"])) {
		fmt.Println([]byte(key.Status.PublicKey))
		fmt.Println([]byte(secret.Data["identity.pub"]))
		return ctrl.Result{},
			fmt.Errorf(
				"Referenced Secret %q does not match the status.publicKey of the %q Key resource. "+
					"This is not automatically reconcilable. "+
					"Please either check and remove the conflicting Secret, or re-create the Key with a non-colliding name. ",
				secretRef, key.GetName())
	}

	log = log.WithValues("repository", key.Spec.RepositoryRef)
	// fetch the accompanying repo for the key
	var repository v1alpha1.Repository
	if err := r.Client.Get(ctx, types.NamespacedName{Name: key.Spec.RepositoryRef, Namespace: req.Namespace}, &repository); err != nil {
		if errors.IsNotFound(err) {
			log.Info("referenced repository does not exist")
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
		log.Error(err, "unexpected error fetching referenced repository")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
		// TODO(stealthybox): This requeue period causes the reconcile to poll for the RepositoryRef.
		// 	It would be an improvement to implement it with exponential backoff, potentially on a Condition similar to Pods failing on a required SecretRef.
		// 	Ideally, we should add a Watch for this particular Repository to a map somewhere or check all of the Keys and queue matching ones when a Repository is queued.
		// 	There are some examples around the internet of subscribing to related objects in a controller.
		// 	This can't be implemented with non-controller, non-blocking ownerRefs because the Repository will still be unintentionally deleted in the background after deleting the Key.
		// 	A re-sync masks this issue.
	}

	// wait for the referenced repository to sync
	if repository.Status.Status != v1alpha1.SyncedStatus {
		r.updateKeyStatus(ctx, &key, v1alpha1.WaitingStatus)
		log.Info("referenced repository not yet synced")
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Repo and Secret are both ready, add finalizer for github-Delete before doing github-Create
	if key.ObjectMeta.DeletionTimestamp.IsZero() &&
		!containsString(key.GetFinalizers(), keyFinalizerName) {
		log.Info("adding finalizer", "name", key.Name)
		if err := r.addFinalizer(ctx, &key); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	var ghKey *github.Key
	var resp *github.Response
	var err error

	createGHKeyAndUpdate := func() (ctrl.Result, error) {
		r.updateKeyStatus(ctx, &key, v1alpha1.CreatingStatus)
		if ghKey, err = r.GitClient.CreateKey(ctx, repository.Spec.Organization, repository.GetName(), &key, &secret); err != nil {
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}

		err = r.updateKeyStatusDetails(ctx, &repository, ghKey, &key)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	if key.Status.GitHubKeyID == 0 {
		log.Info("creating new key in GitHub")
		return createGHKeyAndUpdate()
	}

	ghKey, resp, err = r.GitClient.GetKey(ctx, repository.Spec.Organization, repository.GetName(), key.Status.GitHubKeyID)
	if err != nil && isNotFound(resp) {
		log.Info("expected key not found, creating new key in GitHub", "missingID", key.Status.GitHubKeyID)
		return createGHKeyAndUpdate()
		// note: this can occur on a re-sync despite there being no watch on the GitHub API objects
	} else if err != nil {
		log.Error(err, "error fetching key from GitHub")
		return ctrl.Result{RequeueAfter: requeueAfter}, err
	}

	// recreateGHKey indicates whether the key in GitHub does not match the Key object declaration
	recreateGHKey := (strings.TrimSpace(ghKey.GetKey()) != strings.TrimSpace(key.Status.PublicKey) ||
		ghKey.GetReadOnly() != key.Spec.ReadOnly)
	if recreateGHKey {
		r.updateKeyStatus(ctx, &key, v1alpha1.DeletingStatus)
		err = r.GitClient.DeleteKey(ctx, repository.Spec.Organization, repository.GetName(), key.Status.GitHubKeyID)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueAfter}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

func (r *KeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Key{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
