package wallet

import (
	"fmt"

	"github.com/urfave/cli"

	cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
)

// Register commands
func RegisterCommands(app *cli.App, name string, aliases []string) {
	app.Commands = append(app.Commands, cli.Command{
		Name:    name,
		Aliases: aliases,
		Usage:   "Manage the node wallet",
		Subcommands: []cli.Command{

			{
				Name:      "status",
				Aliases:   []string{"s"},
				Usage:     "Get the node wallet status",
				UsageText: "rocketpool wallet status",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					return getStatus(c)

				},
			},

			{
				Name:      "init",
				Aliases:   []string{"i"},
				Usage:     "Initialize the node wallet",
				UsageText: "rocketpool wallet init [options]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "password, p",
						Usage: "The password to secure the wallet with (if not already set)",
					},
					cli.BoolFlag{
						Name:  "confirm-mnemonic, c",
						Usage: "Automatically confirm the mnemonic phrase",
					},
					cli.StringFlag{
						Name:  "derivation-path, d",
						Usage: "Specify the derivation path for the wallet.\nOmit this flag (or leave it blank) for the default of \"m/44'/60'/0'/0/%d\" (where %d is the index).\nSet this to \"ledgerLive\" to use Ledger Live's path of \"m/44'/60'/%d/0/0\".\nSet this to \"mew\" to use MyEtherWallet's path of \"m/44'/60'/0'/%d\".\nFor custom paths, simply enter them here.",
					},
				},
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Validate flags
					if c.String("password") != "" {
						if _, err := cliutils.ValidateNodePassword("password", c.String("password")); err != nil {
							return err
						}
					}

					// Run
					return initWallet(c)

				},
			},

			{
				Name:      "recover",
				Aliases:   []string{"r"},
				Usage:     "Recover a node wallet from a mnemonic phrase",
				UsageText: "rocketpool wallet recover [options]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "password, p",
						Usage: "The password to secure the wallet with (if not already set)",
					},
					cli.StringFlag{
						Name:  "mnemonic, m",
						Usage: "The mnemonic phrase to recover the wallet from",
					},
					cli.BoolFlag{
						Name:  "skip-validator-key-recovery, k",
						Usage: "Recover the node wallet, but do not regenerate its validator keys",
					},
					cli.StringFlag{
						Name:  "derivation-path, d",
						Usage: "Specify the derivation path for the wallet.\nOmit this flag (or leave it blank) for the default of \"m/44'/60'/0'/0/%d\" (where %d is the index).\nSet this to \"ledgerLive\" to use Ledger Live's path of \"m/44'/60'/%d/0/0\".\nSet this to \"mew\" to use MyEtherWallet's path of \"m/44'/60'/0'/%d\".\nFor custom paths, simply enter them here.",
					},
					cli.UintFlag{
						Name:  "wallet-index, i",
						Usage: "Specify the index to use with the derivation path when recovering your wallet",
						Value: 0,
					},
					cli.StringFlag{
						Name:  "address, a",
						Usage: "If you are recovering a wallet that was not generated by the Smartnode and don't know the derivation path or index of it, enter the address here. The Smartnode will search through its library of paths and indices to try to find it.",
					},
				},
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Validate flags
					if c.String("password") != "" {
						if _, err := cliutils.ValidateNodePassword("password", c.String("password")); err != nil {
							return err
						}
					}
					if c.String("mnemonic") != "" {
						if _, err := cliutils.ValidateWalletMnemonic("mnemonic", c.String("mnemonic")); err != nil {
							return err
						}
					}

					// Run
					return recoverWallet(c)

				},
			},

			{
				Name:      "rebuild",
				Aliases:   []string{"b"},
				Usage:     "Rebuild validator keystores from derived keys",
				UsageText: "rocketpool wallet rebuild",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					return rebuildWallet(c)

				},
			},

			{
				Name:      "test-recovery",
				Aliases:   []string{"t"},
				Usage:     "Test recovering a node wallet without actually generating any of the node wallet or validator key files to ensure the process works as expected",
				UsageText: "rocketpool wallet test-recovery [options]",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "mnemonic, m",
						Usage: "The mnemonic phrase to recover the wallet from",
					},
					cli.BoolFlag{
						Name:  "skip-validator-key-recovery, k",
						Usage: "Recover the node wallet, but do not regenerate its validator keys",
					},
					cli.StringFlag{
						Name:  "derivation-path, d",
						Usage: "Specify the derivation path for the wallet.\nOmit this flag (or leave it blank) for the default of \"m/44'/60'/0'/0/%d\" (where %d is the index).\nSet this to \"ledgerLive\" to use Ledger Live's path of \"m/44'/60'/%d/0/0\".\nSet this to \"mew\" to use MyEtherWallet's path of \"m/44'/60'/0'/%d\".\nFor custom paths, simply enter them here.",
					},
					cli.UintFlag{
						Name:  "wallet-index, i",
						Usage: "Specify the index to use with the derivation path when recovering your wallet",
						Value: 0,
					},
					cli.StringFlag{
						Name:  "address, a",
						Usage: "If you are recovering a wallet that was not generated by the Smartnode and don't know the derivation path or index of it, enter the address here. The Smartnode will search through its library of paths and indices to try to find it.",
					},
				},
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Validate flags
					if c.String("mnemonic") != "" {
						if _, err := cliutils.ValidateWalletMnemonic("mnemonic", c.String("mnemonic")); err != nil {
							return err
						}
					}

					// Run
					return testRecovery(c)

				},
			},

			{
				Name:      "export",
				Aliases:   []string{"e"},
				Usage:     "Export the node wallet in JSON format",
				UsageText: "rocketpool wallet export",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					return exportWallet(c)

				},
			},
			{
				Name:      "set-ens-name",
				Aliases:   []string{"ens"},
				Usage:     "Set a name to the node wallet's ENS reverse record",
				UsageText: "rocketpool wallet set-ens-name",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 1); err != nil {
						return err
					}

					// Run
					return setEnsName(c, c.Args().Get(0))

				},
			},

			{
				Name:      "purge",
				Usage:     fmt.Sprintf("%sDeletes your node wallet, your validator keys, and restarts your Validator Client while preserving your chain data. WARNING: Only use this if you want to stop validating with this machine!%s", colorRed, colorReset),
				UsageText: "rocketpool wallet purge",
				Action: func(c *cli.Context) error {

					// Validate args
					if err := cliutils.ValidateArgCount(c, 0); err != nil {
						return err
					}

					// Run
					return purge(c)

				},
			},
		},
	})
}
