package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/davidmdm/x/xcontext"
	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() (err error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	defer func() {
		if err != nil {
			logger.Error("exiting with error", "error", err.Error())
		}
	}()

	ctx, cancel := xcontext.WithSignalCancelation(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	restCfg, err := func() (*rest.Config, error) {
		if cfg, err := rest.InClusterConfig(); err == nil {
			return cfg, err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube/config"))
	}()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %w", err)
	}

	client, err := k8s.NewClient(restCfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	controller := ctrl.NewController(ctrl.Params{
		Client:      client,
		Logger:      logger,
		Concurrency: runtime.GOMAXPROCS(-1),
	})

	if err := controller.Register(
		ctrl.Entry{
			GroupKind: schema.GroupKind{Group: "joy.nesto.ca", Kind: "Environment"},
			Funcs:     EnvironmentReconciler(),
		},
		ctrl.Entry{
			GroupKind: schema.GroupKind{Group: "joy.nesto.ca", Kind: "Release"},
			Funcs:     ReleaseReconciler(),
		},
	); err != nil {
		return fmt.Errorf("failed to register reconcilers: %w", err)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	e := make(chan error, 2)

	wg.Go(func() {
		logger.Info("starting controller")
		if err := controller.Run(ctx); err != nil {
			e <- err
		}
	})

	wg.Go(func() {
		svr := http.Server{
			Addr: ":3000",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ready" && r.Method == "GET" {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusNotImplemented)
			}),
		}

		serverErr := make(chan error)
		go func() {
			logger.Info("starting server")
			if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		}()

		select {
		case err := <-serverErr:
			e <- fmt.Errorf("failed to start server: %w", err)
			return
		case <-ctx.Done():
			logger.Info("server context canceled", "cause", context.Cause(ctx).Error())
		}

		// TODO: make graceful shutdown period configurable
		ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, fmt.Errorf("exceeded graceful period timeout"))
		defer cancel()

		if err := svr.Shutdown(ctx); err != nil {
			e <- fmt.Errorf("failed to shutdown: %w", err)
		}
	})

	go func() {
		wg.Wait()
		close(e)
	}()

	return <-e
}
