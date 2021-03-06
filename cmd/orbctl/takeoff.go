package main

import (
	"github.com/caos/orbos/cmd/orbctl/cmds"
	"github.com/caos/orbos/internal/start"
	kubernetes2 "github.com/caos/orbos/pkg/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"log"
	"net/http"
	_ "net/http/pprof"
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

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide either one or none")
		}

		ctx, monitor, orbConfig, gitClient, errFunc, err := rv()
		if err != nil {
			return err
		}
		defer func() {
			err = errFunc(err)
		}()

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
		)
	}
	return cmd
}

func StartOrbiter(rv RootValues) *cobra.Command {
	var (
		verbose          bool
		recur            bool
		destroy          bool
		deploy           bool
		ingestionAddress string
		pprof            bool
		cmd              = &cobra.Command{
			Use:   "orbiter",
			Short: "Launch an orbiter",
			Long:  "Ensures a desired state",
		}
	)

	flags := cmd.Flags()
	flags.BoolVar(&recur, "recur", true, "Ensure the desired state continously")
	flags.BoolVar(&deploy, "deploy", true, "Ensure Orbiter deployment continously")
	flags.BoolVar(&pprof, "pprof", false, "Start pprof to analyse memory usage")
	flags.StringVar(&ingestionAddress, "ingestion", "", "Ingestion API address")

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		if recur && destroy {
			return errors.New("flags --recur and --destroy are mutually exclusive, please provide eighter one or none")
		}

		ctx, monitor, orbConfig, gitClient, errFunc, err := rv()
		if err != nil {
			return err
		}
		defer func() {
			err = errFunc(err)
		}()

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

		if pprof {
			go func() {
				log.Println(http.ListenAndServe("localhost:6060", nil))
			}()
		}

		_, err = start.Orbiter(ctx, monitor, orbiterConfig, gitClient, orbConfig, version)
		return err
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

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		_, monitor, orbConfig, _, errFunc, err := rv()
		if err != nil {
			return err
		}
		defer func() {
			err = errFunc(err)
		}()

		return start.Boom(monitor, orbConfig.Path, localmode, version)
	}
	return cmd
}

func StartNetworking(rv RootValues) *cobra.Command {
	var (
		kubeconfig string
		cmd        = &cobra.Command{
			Use:   "networking",
			Short: "Launch a networking operator",
			Long:  "Ensures a desired state of networking for an application",
		}
	)
	flags := cmd.Flags()
	flags.StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig used by zitadel operator")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, monitor, orbConfig, _, errFunc, err := rv()
		if err != nil {
			return err
		}
		defer func() {
			err = errFunc(err)
		}()

		k8sClient, err := kubernetes2.NewK8sClientWithPath(monitor, kubeconfig)
		if err != nil {
			return err
		}

		if k8sClient.Available() {
			return start.Networking(monitor, orbConfig.Path, k8sClient, &version)
		}
		return nil
	}
	return cmd
}
