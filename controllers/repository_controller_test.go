/*
Copyright © 2019 AWS Controller authors

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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"go.hein.dev/github-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Run Controller", func() {
	const timeout = time.Second * 10
	const interval = time.Millisecond * 500

	Context("Run directly without existing job", func() {
		It("Should create successfully", func() {
			Expect(1).To(Equal(1))
		})
	})

	Context("Run a new Repository", func() {
		It("Should create successfully", func() {
			repokey := types.NamespacedName{Name: "test-repo", Namespace: "default"}
			repo := &v1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      repokey.Name,
					Namespace: repokey.Namespace,
				},
				Spec: v1alpha1.RepositorySpec{
					Organization: "awsctrl",
				},
			}
			Expect(k8sClient.Create(context.Background(), repo)).Should(Succeed())

			By("Describing Repository Finalizers")
			Eventually(func() bool {
				r := &v1alpha1.Repository{}
				k8sClient.Get(context.Background(), repokey, r)
				return len(r.GetFinalizers()) == 1
			}, timeout, interval).Should(BeTrue())

			// By("Describing Creating Repo")
			// Eventually(func() bool {
			// 	r := &v1alpha1.Repository{}
			// 	k8sClient.Get(context.Background(), repokey, r)
			// 	return r.Status.Status == v1alpha1.CreatingStatus
			// }, timeout, interval).Should(BeTrue())

			By("Describing Getting URL")
			Eventually(func() bool {
				r := &v1alpha1.Repository{}
				k8sClient.Get(context.Background(), repokey, r)
				return r.Status.URL == "https://github.com/awsctrl/test-repo"
			}, timeout, interval).Should(BeTrue())

			By("Describing Getting Synced Status")
			Eventually(func() bool {
				r := &v1alpha1.Repository{}
				k8sClient.Get(context.Background(), repokey, r)
				return r.Status.Status == v1alpha1.SyncedStatus
			}, timeout, interval).Should(BeTrue())

			By("Describing Getting the final status updates")
			Eventually(func() bool {
				r := &v1alpha1.Repository{}
				k8sClient.Get(context.Background(), repokey, r)
				return r.Status.WatchersCount == 0
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(context.Background(), repo)).Should(Succeed())

			By("Describing deletion state")
			Eventually(func() bool {
				r := &v1alpha1.Repository{}
				k8sClient.Get(context.Background(), repokey, r)
				return len(r.GetFinalizers()) == 0
			}, timeout, interval).Should(BeTrue())
		})
	})
})
