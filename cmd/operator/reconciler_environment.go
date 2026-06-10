package main

import (
	"context"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nestoca/joy/api/v1alpha1"
)

func EnvironmentReconciler() ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			var (
				client   = ctrl.Client(ctx)
				envCache = ctrl.CacheFromEvent[v1alpha1.Environment](ctx, event)
				nsIntf   = k8s.TypedInterface[corev1.Namespace](client.Dynamic, schema.GroupVersionResource{
					Version:  "v1",
					Resource: "namespaces",
				})
			)

			_, err := envCache.Get(event.Name)
			if err != nil {
				return ctrl.Result{}, err
			}

			_, _ = nsIntf.Apply(ctx, nil, metav1.ApplyOptions{})

			return ctrl.Result{}, nil
		},
	}
}
