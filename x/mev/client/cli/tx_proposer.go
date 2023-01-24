package cli

import (
	"fmt"

	"cosmossdk.io/x/mev/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func CmdRegisterProposer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-proposer <builder_module_proposer_key.json>",
		Short: "Register a proposer for off-chain block building",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return fmt.Errorf("get client context: %w", err)
			}

			proposerKey, err := types.LoadOrGenProposerKey(args[0], clientCtx.Codec)
			if err != nil {
				return fmt.Errorf("load proposer key: %w", err)
			}

			address := sdk.AccAddress(proposerKey.PubKey.Address())
			operatorAddr := sdk.ValAddress(clientCtx.FromAddress)
			operatorPubKey, err := clientCtx.Keyring.KeyByAddress(operatorAddr)
			if err != nil {
				return fmt.Errorf("get operator pubkey: %w", err)
			}

			pubkey, err := codectypes.NewAnyWithValue(proposerKey.PubKey)
			if err != nil {
				return fmt.Errorf("convert proposer pubkey: %w", err)
			}

			msg := &types.MsgRegisterProposer{
				Address:         address.String(),
				Pubkey:          pubkey,
				OperatorAddress: operatorAddr.String(),
				OperatorPubkey:  operatorPubKey.PubKey,
			}

			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("validate register proposer msg: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
