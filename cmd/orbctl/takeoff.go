package main

import (
	"github.com/caos/orbos/cmd/orbctl/cmds"
	"github.com/caos/orbos/internal/controller"
	"github.com/caos/orbos/internal/start"
	kubernetes2 "github.com/caos/orbos/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TakeoffCommand(getRv GetRootValues) *cobra.Command {
	var (
		verbose          bool
		recur            bool
		destroy          bool
		deploy           bool
		kubeconfig       string
		ingestionAddress string
		gitOpsBoom       bool
		gitOpsNetworking bool
		cmd              = &cobra.Command{
			Use:   "takeoff",
			Short: "Launch an orbiter",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&recur, "recur", false, "Ensure the desired state continously")
	flags.BoolVar(&deploy, "deploy", true, "Ensure Orbiter and Boom deployments continously")
	flags.BoolVar(&gitOpsBoom, "gitops-boom", false, "Ensure Boom runs in gitops mode")
	flags.BoolVar(&gitOpsNetworking, "gitops-networking", false, "Ensure Networking-operator runs in gitops mode")
	flags.StringVar(&ingestionAddress, "ingestion", "", "Ingestion API address")
	flags.StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig for boom deployment")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide eighter one or none")
		}

		rv, err := getRv()
		if err != nil {
			return err
		}
		defer func() {
			err = rv.ErrFunc(err)
		}()

		orbConfig := rv.OrbConfig
		gitClient := rv.GitClient
		ctx := rv.Ctx

		return cmds.Takeoff(
			monitor,
			ctx,
			orbConfig,
			gitClient,
			recur,
			destroy,
			deploy,
			verbose,
			ingestionAddress,
			version,
			gitCommit,
			kubeconfig,
			gitOpsBoom,
			gitOpsNetworking,
		)
	}
	return cmd
}

func StartOrbiter(getRv GetRootValues) *cobra.Command {
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
	flags.BoolVar(&deploy, "deploy", true, "Ensure Orbiter deployment continously")
	flags.StringVar(&ingestionAddress, "ingestion", "", "Ingestion API address")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide eighter one or none")
		}

		rv, err := getRv()
		if err != nil {
			return err
		}
		defer func() {
			err = rv.ErrFunc(err)
		}()

		monitor := rv.Monitor
		orbConfig := rv.OrbConfig
		gitClient := rv.GitClient
		ctx := rv.Ctx
		if err := gitClient.Configure(orbConfig.URL, []byte(orbConfig.Repokey)); err != nil {
			return err
		}

		orbiterConfig := &start.OrbiterConfig{
			Recur:            recur,
			Destroy:          destroy,
			Deploy:           deploy,
			Verbose:          verbose,
			Version:          version,
			OrbConfigPath:    orbConfig.Path,
			GitCommit:        gitCommit,
			IngestionAddress: ingestionAddress,
		}

		_, err = start.Orbiter(ctx, monitor, orbiterConfig, gitClient, orbConfig, version)
		return err
	}
	return cmd
}

func StartBoom(getRv GetRootValues) *cobra.Command {
	var (
		localmode  bool
		gitOpsMode bool
		cmd        = &cobra.Command{
			Use:   "boom",
			Short: "Launch a boom",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&localmode, "localmode", false, "Local mode for boom")
	flags.BoolVar(&gitOpsMode, "gitops", false, "defines if the operator should run in gitops mode")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {

		rv, err := getRv()
		if err != nil {
			return err
		}
		defer func() {
			err = rv.ErrFunc(err)
		}()

		monitor := rv.Monitor
		orbConfig := rv.OrbConfig

		if gitOpsMode {
			return start.Boom(monitor, orbConfig.Path, localmode, version)
		} else {
			return controller.Start(monitor, version, "/boom", rv.MetricsAddr, controller.Boom)
		}
	}
	return cmd
}

func StartNetworking(getRv GetRootValues) *cobra.Command {
	var (
		gitOpsMode bool
		kubeconfig string
		cmd        = &cobra.Command{
			Use:   "networking",
			Short: "Launch a networking operator",
			Long:  "Ensures a desired state of networking for an application",
		}
	)
	flags := cmd.Flags()
	flags.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig used by zitadel operator")
	flags.BoolVar(&gitOpsMode, "gitops", false, "defines if the operator should run in gitops mode")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {

		rv, err := getRv()
		if err != nil {
			return err
		}
		defer func() {
			err = rv.ErrFunc(err)
		}()

		monitor := rv.Monitor
		orbConfig := rv.OrbConfig

		if gitOpsMode {
			k8sClient, err := kubernetes2.NewK8sClientWithPath(monitor, kubeconfig)
			if err != nil {
				return err
			}

			if k8sClient.Available() {
				return start.Networking(monitor, orbConfig.Path, k8sClient, &version)
			}
		} else {
			return controller.Start(monitor, version, "/boom", rv.MetricsAddr, controller.Networking)
		}
		return nil
	}
	return cmd
}
