package main

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestMain(m *testing.M) {
	if run, _ := strconv.ParseBool(os.Getenv("RUN_OPERATOR_TESTS")); !run {
		os.Exit(0)
	}

	if err := withStdio(exec.Command("kind", "delete", "cluster", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("kind", "create", "cluster", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("docker", "build", "--tag=joy-operator:test", "../..")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(exec.Command("kind", "load", "docker-image", "joy-operator:test", "--name=joy-operator")).Run(); err != nil {
		panic(err)
	}

	if err := withStdio(
		exec.Command(
			"helm",
			"install",
			"joy-operator",
			"../../chart",
			"--set=image=joy-operator",
			"--set=version=test",
			"--set=argocd.namespace=default",
		),
	).Run(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func withStdio(cmd *exec.Cmd) *exec.Cmd {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}
