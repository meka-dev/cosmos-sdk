package cli

import (
	"context"
	"errors"
	"os"

	"cosmossdk.io/x/mev/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk_crypto_types "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func CmdListBuilder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-builders",
		Short: "List all builders",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryBuildersRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.Builders(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowBuilder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-builder [address]",
		Short: "Shows a builder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argAddress := args[0]

			params := &types.QueryBuilderRequest{Address: argAddress}

			res, err := queryClient.Builder(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdExportJSONKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-json-key [address] [output-file]",
		Short: "Exports a key in JSON format, to be used by Builder APIs to sign",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			address, err := sdk_types.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			keyRecord, err := clientCtx.Keyring.KeyByAddress(address)
			if err != nil {
				return err
			}

			privKey, ok := keyRecord.GetLocal().PrivKey.GetCachedValue().(sdk_crypto_types.PrivKey)
			if !ok {
				return errors.New("unable to cast any to cryptotypes.PrivKey")
			}

			out, err := clientCtx.Codec.MarshalInterfaceJSON(privKey)
			if err != nil {
				return err
			}

			if err := os.WriteFile(args[1], out, 0600); err != nil {
				return err
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddKeyringFlags(cmd.Flags())

	return cmd
}
