package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	cometbft_rpc_core_types "github.com/cometbft/cometbft/rpc/core/types"
	sdk_client "github.com/cosmos/cosmos-sdk/client"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	sdk_x_auth_types "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type Node interface {
	CurrentHeight(ctx context.Context) (int64, error)
	AccountNumber(ctx context.Context, addr sdk_types.AccAddress) (uint64, error)
	Proposer(ctx context.Context, height int64, addr string) (*sdk_x_builder_types.QueryProposerResponse, error)
	AccountAtHeight(ctx context.Context, height int64, addr string) (*sdk_x_auth_types.BaseAccount, error)
	VerifyTxInclusion(ctx context.Context, height int64, txHash []byte) error
	BroadcastTxAsync(ctx context.Context, tx []byte) (*cometbft_rpc_core_types.ResultBroadcastTx, error)
}

type DirectNode struct {
	baseurl string
}

func NewDirectNode(baseurl string) (*DirectNode, error) {
	u, err := url.Parse(baseurl)
	if err != nil {
		return nil, err
	}

	u.Path = ""
	u.RawQuery = ""

	n := &DirectNode{
		baseurl: u.String(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := n.Ping(ctx); err != nil {
		return nil, err
	}

	return n, nil
}

func (n *DirectNode) Ping(ctx context.Context) error {
	var res struct {
		Result struct {
			SyncInfo struct {
				CatchingUp bool `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	validate := func() error {
		if res.Result.SyncInfo.CatchingUp {
			return fmt.Errorf("catching up")
		}
		return nil
	}

	if err := n.do(ctx, "GET", "/status", nil, &res, validate); err != nil {
		return err
	}

	return nil
}

func (n *DirectNode) CurrentHeight(ctx context.Context) (int64, error) {
	var res struct {
		Result struct {
			SyncInfo struct {
				CurrentHeight string `json:"latest_block_height"`
				CatchingUp    bool   `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	validate := func() error {
		if res.Result.SyncInfo.CatchingUp {
			return fmt.Errorf("catching up")
		}
		return nil
	}

	if err := n.do(ctx, "GET", "/status", nil, &res, validate); err != nil {
		return 0, err
	}

	h, err := strconv.ParseInt(res.Result.SyncInfo.CurrentHeight, 10, 64)
	if err != nil {
		return 0, err
	}

	return h, nil
}

func (n *DirectNode) AccountNumber(ctx context.Context, addr sdk_types.AccAddress) (uint64, error) {
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return 0, err
	}

	clientContext := sdk_client.Context{}.
		WithNodeURI(n.baseurl).
		WithClient(rpcClient).
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithLegacyAmino(encodingConfig.Amino).
		WithAccountRetriever(sdk_x_auth_types.AccountRetriever{})

	account, err := clientContext.AccountRetriever.GetAccount(clientContext, addr)
	if err != nil {
		return 0, err
	}

	return account.GetAccountNumber(), nil
}

func (n *DirectNode) Proposer(ctx context.Context, height int64, proposerAddr string) (*sdk_x_builder_types.QueryProposerResponse, error) {
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return nil, fmt.Errorf("construct client: %w", err)
	}

	clientContext := sdk_client.Context{}.
		WithNodeURI(n.baseurl).
		WithClient(rpcClient).
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithLegacyAmino(encodingConfig.Amino).
		WithAccountRetriever(sdk_x_auth_types.AccountRetriever{}).
		WithHeight(height)

	client := sdk_x_builder_types.NewQueryClient(clientContext)

	res, err := client.Proposer(ctx, &sdk_x_builder_types.QueryProposerRequest{
		Address: proposerAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("execute proposer query: %w", err)
	}

	return res, nil
}

func (n *DirectNode) ProposersAtHeight(ctx context.Context, height int64) ([]sdk_x_builder_types.Proposer, error) { // TODO: fix height
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return nil, fmt.Errorf("construct client: %w", err)
	}

	clientContext := sdk_client.Context{}.
		WithNodeURI(n.baseurl).
		WithClient(rpcClient).
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithLegacyAmino(encodingConfig.Amino).
		WithAccountRetriever(sdk_x_auth_types.AccountRetriever{})

	client := sdk_x_builder_types.NewQueryClient(clientContext)

	res, err := client.Proposers(ctx, &sdk_x_builder_types.QueryProposersRequest{
		// Pagination: &query.PageRequest{}, // TODO: pagination
	})
	if err != nil {
		return nil, fmt.Errorf("execute proposer query: %w", err)
	}

	return res.GetProposers(), nil
}

func (n *DirectNode) VerifyTxInclusion(ctx context.Context, height int64, txHash []byte) error {
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return fmt.Errorf("construct client: %w", err)
	}

	// TODO: this doesn't let us distinguish "tx not found" from other e.g. network errors
	// TODO: probably need to fetch the block for height and manually search the txs
	res, err := rpcClient.Tx(ctx, txHash, false)
	if err != nil {
		return fmt.Errorf("query for tx: %w", err)
	}

	if res.Height != height {
		return fmt.Errorf("tx %x found at height %d, want %d", txHash, res.Height, height)
	}

	return nil
}

func (n *DirectNode) AccountAtHeight(ctx context.Context, height int64, addr string) (*sdk_x_auth_types.BaseAccount, error) {
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return nil, fmt.Errorf("construct client: %w", err)
	}

	clientContext := sdk_client.Context{}.
		WithNodeURI(n.baseurl).
		WithClient(rpcClient).
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithLegacyAmino(encodingConfig.Amino).
		WithAccountRetriever(sdk_x_auth_types.AccountRetriever{}).
		WithHeight(height) // height set here

	client := sdk_x_auth_types.NewQueryClient(clientContext)

	res, err := client.AccountInfo(ctx, &sdk_x_auth_types.QueryAccountInfoRequest{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("execute account query: %w", err)
	}

	info := res.GetInfo()
	if info == nil {
		return nil, fmt.Errorf("nil info in response")
	}

	// BUG: Validate segfaults because inner pubkey is invalid -- bug in the SDK
	//if err := info.Validate(); err != nil {
	//	return nil, fmt.Errorf("invalid info in response: %w", err)
	//}

	return info, nil
}

func (n *DirectNode) BroadcastTxAsync(ctx context.Context, tx []byte) (*cometbft_rpc_core_types.ResultBroadcastTx, error) {
	rpcClient, err := sdk_client.NewClientFromNode(n.baseurl)
	if err != nil {
		return nil, fmt.Errorf("construct client: %w", err)
	}

	return rpcClient.BroadcastTxAsync(ctx, tx)
}

func (n *DirectNode) do(ctx context.Context, method, path string, body []byte, res any, validate func() error) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, n.baseurl+path, bytes.NewReader(body))
	if err != nil {
		return err
	}

	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, httpRes.Body)
		httpRes.Body.Close()
	}()

	if err := json.NewDecoder(httpRes.Body).Decode(res); err != nil {
		return err
	}

	if validate != nil {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

//
//
//

type NodeWithCache struct {
	*DirectNode

	mtx       sync.Mutex
	proposers map[int64]map[string]*sdk_x_builder_types.QueryProposerResponse
	account   map[heightAddress]*sdk_x_auth_types.BaseAccount
}

type heightAddress struct {
	height  int64
	address string
}

func (c *NodeWithCache) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ready := func() bool {
				c.mtx.Lock()
				defer c.mtx.Unlock()
				return c.proposers != nil && c.account != nil
			}(); ready {
				return nil
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *NodeWithCache) RefreshProposers(ctx context.Context) error {
	currentHeight, err := n.DirectNode.CurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("get current height: %w", err)
	}

	nextgen := map[int64]map[string]*sdk_x_builder_types.QueryProposerResponse{}
	for h := currentHeight - 1; h <= currentHeight; h++ {
		ps, err := n.DirectNode.ProposersAtHeight(ctx, h)
		if err != nil {
			return fmt.Errorf("get proposers (current height %d, target height %d): %w", currentHeight, h, err)
		}

		index := map[string]*sdk_x_builder_types.QueryProposerResponse{}
		for _, p := range ps {
			res, err := n.DirectNode.Proposer(ctx, h, p.Address)
			if err != nil {
				return fmt.Errorf("get proposer %s (current height %d, target height %d): %w", p.Address, currentHeight, h, err)
			}
			index[p.Address] = res
		}

		nextgen[h] = index
	}

	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.proposers = nextgen

	return nil
}

func (n *NodeWithCache) Proposer(ctx context.Context, height int64, proposerAddr string) (*sdk_x_builder_types.QueryProposerResponse, error) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	proposers, ok := n.proposers[height]
	if !ok {
		return nil, fmt.Errorf("proposers for height %d not cached", height)
	}

	p, ok := proposers[proposerAddr]
	if !ok {
		return nil, fmt.Errorf("proposer %s does not exist in cached height %d", proposerAddr, height)
	}

	return p, nil
}

func (n *NodeWithCache) RefreshAccounts(ctx context.Context, addrs ...string) error {
	currentHeight, err := n.DirectNode.CurrentHeight(ctx)
	if err != nil {
		return fmt.Errorf("get current height: %w", err)
	}

	nextgen := map[heightAddress]*sdk_x_auth_types.BaseAccount{}
	for _, addr := range addrs {
		info, err := n.DirectNode.AccountAtHeight(ctx, currentHeight, addr)
		if err != nil {
			return fmt.Errorf("error fetching account info (%s): %w", addr, err)
		}

		nextgen[heightAddress{
			height:  currentHeight,
			address: addr,
		}] = info
	}

	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.account = nextgen

	return nil
}

func (n *NodeWithCache) AccountAtHeight(ctx context.Context, height int64, addr string) (*sdk_x_auth_types.BaseAccount, error) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	key := heightAddress{height: height, address: addr}
	info, ok := n.account[key]
	if !ok {
		return nil, fmt.Errorf("account (%s) at height (%d) not cached", addr, height)
	}

	return info, nil
}
