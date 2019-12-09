package main

import (
	"os"

	"github.com/caos/orbiter/internal/core/secret"
	"github.com/spf13/cobra"
)

func readSecretCommand(rv rootValues) *cobra.Command {

	return &cobra.Command{
		Use:   "readsecret [name]",
		Short: "Decrypt and print to stdout",
		Args:  cobra.ExactArgs(1),
		Example: `
mkdir -p ~/.kube
orbctl --repourl git@github.com:example/my-orb.git \
       --repokey-file ~/.ssh/my-orb --masterkey 'my very secret key'
       readsecret myorbk8s_kubeconfig > ~/.kube/config`,
		RunE: func(cmd *cobra.Command, args []string) error {

			_, logger, gitClient, _, _, mk, err := rv(false)
			if err != nil {
				return err
			}

			if err := gitClient.Clone(); err != nil {
				panic(err)
			}

			sec, err := gitClient.Read("secrets.yml")
			if err != nil {
				panic(err)
			}

			if err := secret.New(logger, sec, args[0], mk).Read(os.Stdout); err != nil {
				panic(err)
			}
			return nil
		},
	}
}
