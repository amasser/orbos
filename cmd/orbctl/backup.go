package main

import (
	"github.com/caos/orbos/internal/api"
	"github.com/caos/orbos/internal/start"
	kubernetes2 "github.com/caos/orbos/pkg/kubernetes"
	"github.com/spf13/cobra"
)

func BackupCommand(getRv GetRootValues) *cobra.Command {
	var (
		backup string
		cmd    = &cobra.Command{
			Use:   "backup",
			Short: "Instant backup",
			Long:  "Instant backup",
		}
	)

	flags := cmd.Flags()
	flags.StringVar(&backup, "backup", "", "Name used for backup folder")

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
		gitClient := rv.GitClient

		if err := orbConfig.IsConnectable(); err != nil {
			return err
		}

		if err := gitClient.Configure(orbConfig.URL, []byte(orbConfig.Repokey)); err != nil {
			return err
		}

		if err := gitClient.Clone(); err != nil {
			return err
		}

		found, err := api.ExistsDatabaseYml(gitClient)
		if err != nil {
			return err
		}
		if found {
			kubeconfigs, err := start.GetKubeconfigs(monitor, gitClient, orbConfig, version)
			if err != nil {
				return err
			}
			for _, kubeconfig := range kubeconfigs {
				k8sClient := kubernetes2.NewK8sClient(monitor, &kubeconfig)
				if k8sClient.Available() {
					if err := start.DatabaseBackup(monitor, orbConfig.Path, k8sClient, backup, &version); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	return cmd
}
