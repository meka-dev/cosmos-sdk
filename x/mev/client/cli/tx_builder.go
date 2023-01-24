package cli

import (
	"cosmossdk.io/x/mev/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func CmdRegisterBuilder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-builder",
		Short: "Register an off-chain builder",
		Args:  cobra.ExactArgs(0),

		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubKey, err := clientCtx.Keyring.Key(clientCtx.From)
			if err != nil {
				pubKey, err = clientCtx.Keyring.KeyByAddress(clientCtx.FromAddress)
				if err != nil {
					return err
				}
			}

			moniker, err := cmd.Flags().GetString("moniker")
			if err != nil {
				return err
			}

			builderAPIVersion, err := cmd.Flags().GetString("builder-api-version")
			if err != nil {
				return err
			}

			builderAPIURL, err := cmd.Flags().GetString("builder-api-url")
			if err != nil {
				return err
			}

			securityContact, err := cmd.Flags().GetString("security-contact")
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterBuilder{
				Address:           clientCtx.From,
				Pubkey:            pubKey.PubKey,
				Moniker:           moniker,
				BuilderApiVersion: builderAPIVersion,
				BuilderApiUrl:     builderAPIURL,
				SecurityContact:   securityContact,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("moniker", "", "Human readable name of this builder")
	cmd.Flags().String("builder-api-version", "", "Builder API version string")
	cmd.Flags().String("builder-api-url", "", "Builder API URL")
	cmd.Flags().String("security-contact", "", "Security contact email")

	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	_ = cmd.MarkFlagRequired("moniker")
	_ = cmd.MarkFlagRequired("builder-api-version")
	_ = cmd.MarkFlagRequired("builder-api-url")
	_ = cmd.MarkFlagRequired("security-contact")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdEditBuilder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit-builder",
		Short: "Edit a builder",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubKey, err := clientCtx.Keyring.KeyByAddress(clientCtx.FromAddress)
			if err != nil {
				return err
			}

			moniker, err := cmd.Flags().GetString("moniker")
			if err != nil {
				return err
			}

			builderAPIVersion, err := cmd.Flags().GetString("builder-api-version")
			if err != nil {
				return err
			}

			builderAPIURL, err := cmd.Flags().GetString("builder-api-url")
			if err != nil {
				return err
			}

			securityContact, err := cmd.Flags().GetString("security-contact")
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterBuilder{
				Address:           clientCtx.From,
				Pubkey:            pubKey.PubKey,
				Moniker:           moniker,
				BuilderApiVersion: builderAPIVersion,
				BuilderApiUrl:     builderAPIURL,
				SecurityContact:   securityContact,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("moniker", "", "Human readable name of this builder")
	cmd.Flags().String("builder-api-version", "", "Builder API version string")
	cmd.Flags().String("builder-api-url", "", "Builder API URL")
	cmd.Flags().String("security-contact", "", "Security contact email")

	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	_ = cmd.MarkFlagRequired("moniker")
	_ = cmd.MarkFlagRequired("builder-api-version")
	_ = cmd.MarkFlagRequired("builder-api-url")
	_ = cmd.MarkFlagRequired("security-contact")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
