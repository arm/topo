package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/ssh/sshconfig"
	"github.com/spf13/cobra"
)

var (
	privateKeyPath string
	keyType        string
)

var setupKeysCmd = &cobra.Command{
	Use:   "setup-keys",
	Short: "Generate SSH keys for the target and install the public key on the target host",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)
		targetSlug := dest.Slugify()
		if privateKeyPath == "" {
			privateKeyPath, err = setupkeys.GetDefaultPrivateKeyPath(targetSlug)
			if err != nil {
				return err
			}
		}

		parsedKeyType, err := setupkeys.ParseKeyType(keyType)
		if err != nil {
			return err
		}

		seq, err := setupkeys.NewKeySetup(dest, privateKeyPath, parsedKeyType)
		if err != nil {
			return err
		}

		err = seq.Run(os.Stdout)
		if err != nil {
			return err
		}

		directives := []sshconfig.SSHConfigDirective{
			sshconfig.NewDirectiveIdentityFile(privateKeyPath),
			sshconfig.NewDirective("IdentitiesOnly", "yes"),
		}

		return sshconfig.CreateOrModifySSHConfig(targetArg, targetSlug, directives)
	},
}

func init() {
	addTargetFlag(setupKeysCmd)
	setupKeysCmd.Flags().StringVar(&privateKeyPath, "key-path", "", "Specify the SSH path where the generated key pair will be stored. Default directory: ~/.ssh. Default public key file name: id_ed25519_topo_<target>.pub)")
	setupKeysCmd.Flags().StringVar(&keyType, "key-type", "ed25519", fmt.Sprintf("Specify the type of SSH key to generate. Supported types: %s, %s. Default: %s", setupkeys.KeyTypeED25519, setupkeys.KeyTypeRSA, setupkeys.KeyTypeED25519))
	rootCmd.AddCommand(setupKeysCmd)
}
