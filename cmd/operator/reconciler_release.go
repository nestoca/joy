package main

import (
	"context"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/nestoca/joy/cmd/operator/argocd"
)

func ReleaseReconciler() ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			var (
				releaseCache = ctrl.CacheFromEvent[v1alpha1.Release](ctx, event)
				envCache     = ctrl.Cache[v1alpha1.Environment](ctx, schema.GroupKind{Group: "joy.nesto.ca", Kind: "Environment"}, "")
				client       = ctrl.Client(ctx)
				appIntf      = k8s.TypedInterface[argocd.Application](client.Dynamic, schema.GroupVersionResource{
					Group:    "argoproj.io",
					Version:  "v1alpha1",
					Resource: "applications",
				})
			)

			release, err := releaseCache.Get(event.Name)
			if err != nil {
				return ctrl.Result{}, err
			}

			env, err := envCache.Get(release.Name)
			if err != nil {
				return ctrl.Result{}, err
			}

			// do something with env???
			_ = env

			// do something with application client?
			_ = appIntf

			return ctrl.Result{}, nil
		},
	}
}
