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

package main

import (
	"context"
	"flag"
	"os"

	githubv1alpha1 "go.hein.dev/github-controller/api/v1alpha1"
	"go.hein.dev/github-controller/controllers"
	"go.hein.dev/github-controller/git"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlv1alpha1 "sigs.k8s.io/controller-runtime/pkg/api/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ctrlv1alpha1.AddToScheme(scheme))
	utilruntime.Must(githubv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var port int
	var metricsAddr string
	var enableLeaderElection bool
	var actualDelete bool
	var config string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.IntVar(&port, "port", 9443, "The port the server binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&actualDelete, "actual-delete", false, "default: false; when true it will actually delete repos when the manifest is deleted.")
	flag.StringVar(&config, "config", "", "Config file for loading configurations.")

	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	options := ctrl.Options{Scheme: scheme}
	var err error

	if config != "" {
		options, err = ctrl.NewOptionsFromComponentConfig(scheme, config, &ctrlv1alpha1.DefaultControllerConfiguration{})
		if err != nil {
			setupLog.Error(err, "unable to update options")
			os.Exit(1)
		}
	}

	gitclient := git.New(context.Background(), os.Getenv("GITHUB_AUTH_TOKEN"))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.RepositoryReconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("Repository"),
		Scheme:       mgr.GetScheme(),
		GitClient:    gitclient,
		ActualDelete: actualDelete,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Repository")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
