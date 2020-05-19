package main

import (
	"github.com/caos/orbos/internal/operator/boom/api"
	"github.com/caos/orbos/internal/operator/orbiter"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/kubernetes"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/orb"
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/start"
	"github.com/caos/orbos/mntr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strings"
)

func TakeoffCommand(rv RootValues) *cobra.Command {

	var (
		verbose          bool
		recur            bool
		destroy          bool
		deploy           bool
		kubeconfig       string
		ingestionAddress string
		cmd              = &cobra.Command{
			Use:   "takeoff",
			Short: "Launch an orbiter",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&recur, "recur", false, "Ensure the desired state continously")
	flags.BoolVar(&deploy, "deploy", true, "Ensure Orbiter and Boom deployments continously")
	flags.StringVar(&ingestionAddress, "ingestion", "", "Ingestion API address")
	flags.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig for boom deployment")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		orbiterIncluded := false

		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide eighter one or none")
		}

		ctx, monitor, gitClient, orbFile, errFunc := rv()
		if errFunc != nil {
			return errFunc(cmd)
		}

		if len(gitClient.Read("orbiter.yml")) > 0 {
			if err := start.Orbiter(ctx, monitor, recur, destroy, deploy, verbose, version, gitClient, orbFile, gitCommit, ingestionAddress); err != nil {
				return err
			}
			orbiterIncluded = true

			orbTree, err := orbiter.Parse(gitClient, "orbiter.yml")
			if err != nil {
				return errors.New("Failed to parse orbiter.yml")
			}

			orbDef, err := orb.ParseDesiredV0(orbTree[0])
			if err != nil {
				return errors.New("Failed to parse orbiter.yml")
			}

			for clustername, _ := range orbDef.Clusters {
				path := strings.Join([]string{"orbiter", clustername, "kubeconfig"}, ".")
				secretFunc := func(operator string) secret.Func {
					if operator == "boom" {
						return api.SecretFunc(orbFile)
					} else if operator == "orbiter" {
						return orb.SecretsFunc(orbFile)
					}
					return nil
				}

				value, err := secret.Read(
					monitor,
					gitClient,
					secretFunc,
					path)
				if err != nil || value == "" {
					return errors.New("Failed to get kubeconfig")
				}
				monitor.Info("Read kubeconfig for boom deployment")

				if err := deployBoom(monitor, value); err != nil {
					return err
				}
			}
		}

		if !orbiterIncluded && len(gitClient.Read("boom.yml")) > 0 {
			if kubeconfig == "" {
				return errors.New("Error to deploy BOOM as no kubeconfig is provided")
			}

			if err := deployBoom(monitor, kubeconfig); err != nil {
				return err
			}
		}
		return nil
	}
	return cmd
}

func deployBoom(monitor mntr.Monitor, kubeconfig string) error {
	k8sClient := kubernetes.NewK8sClient(monitor, &kubeconfig)

	if k8sClient.Available() {
		if err := kubernetes.EnsureBoomArtifacts(monitor, k8sClient, version); err != nil {
			monitor.Info("failed to deploy boom into k8s-cluster")
			return err
		}
		monitor.Info("Deployed boom")
	} else {
		monitor.Info("Failed to connect to k8s")
	}
	return nil
}

func StartOrbiter(rv RootValues) *cobra.Command {
	var (
		verbose          bool
		recur            bool
		destroy          bool
		deploy           bool
		ingestionAddress string
		cmd              = &cobra.Command{
			Use:   "orbiter",
			Short: "Launch an orbiter",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&recur, "recur", true, "Ensure the desired state continously")
	flags.BoolVar(&deploy, "deploy", true, "Ensure Orbiter and Boom deployments continously")
	flags.StringVar(&ingestionAddress, "ingestion", "", "Ingestion API address")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide eighter one or none")
		}

		ctx, monitor, gitClient, orbFile, errFunc := rv()
		if errFunc != nil {
			return errFunc(cmd)
		}

		return start.Orbiter(ctx, monitor, recur, destroy, deploy, verbose, version, gitClient, orbFile, gitCommit, ingestionAddress)
	}
	return cmd
}

func StartBoom(rv RootValues) *cobra.Command {
	var (
		localmode bool
		cmd       = &cobra.Command{
			Use:   "boom",
			Short: "Launch a boom",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&localmode, "localmode", false, "Local mode for boom")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, monitor, _, orbFile, errFunc := rv()
		if errFunc != nil {
			return errFunc(cmd)
		}

		return start.Boom(monitor, orbFile, localmode)
	}
	return cmd
}
