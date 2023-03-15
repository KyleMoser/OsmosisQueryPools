package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkQuery "github.com/cosmos/cosmos-sdk/types/query"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	osmosis "github.com/osmosis-labs/osmosis/v15/app"
	osmosisParams "github.com/osmosis-labs/osmosis/v15/app/params"
	gammBalancer "github.com/osmosis-labs/osmosis/v15/x/gamm/pool-models/balancer"
	"github.com/osmosis-labs/osmosis/v15/x/gamm/types"
)

func main() {
	encodingConfig := osmosis.MakeEncodingConfig()
	chainID := "osmosis-1"
	rpc := "https://rpc.osmosis.zone:443"
	homeDir := "/home" //doesn't really matter unless you're signing TXs
	key := "default"   //doesn't really matter unless signing TXs

	queryClient, err := GetOsmosisTxClient(encodingConfig, chainID, rpc, homeDir, "test", key)
	if err != nil {
		os.Exit(1)
	}

	//omitting this implies the current chain height
	//queryClient = queryClient.WithHeight(blockHeight)

	whitelistedPools := map[uint64]struct{}{}
	whitelistedPools[1] = struct{}{}

	gammPools, err := QueryGammPools(queryClient, whitelistedPools, true)
	if err != nil {
		os.Exit(1)
	} else {
		for _, p := range gammPools {
			fmt.Printf("Woot, a pool we want! %+v\n", p)
		}
	}
}

func QueryGammPools(
	ctx client.Context,
	whitelist map[uint64]struct{}, //Pools whitelisted by ID
	useWhitelist bool, //If false, all pools will be included
) (
	map[uint64]*gammBalancer.Pool, //Map of pool ID to the pool
	error,
) {
	pools := map[uint64]*gammBalancer.Pool{}
	paginator := sdkQuery.PageRequest{}
	queryClient := types.NewQueryClient(ctx)

	for paginate := true; paginate; { // break out of the loop when there is no next key
		resp, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{Pagination: &paginator})
		if err != nil {
			return nil, err
		}

		for _, osmoPool := range resp.Pools {
			var osmoPoolVal types.CFMMPoolI
			ctx.InterfaceRegistry.UnpackAny(osmoPool, &osmoPoolVal)
			gammPool, isBalancerPool := osmoPoolVal.(*gammBalancer.Pool)
			if !isBalancerPool {
				continue
			}

			if !useWhitelist || whitelist == nil {
				pools[gammPool.Id] = gammPool
			} else if _, ok := whitelist[gammPool.Id]; ok {
				pools[gammPool.Id] = gammPool
			}

			paginate = resp.Pagination.NextKey != nil && len(resp.Pagination.NextKey) > 0
			paginator.Key = resp.Pagination.NextKey
		}
	}

	return pools, nil
}

// chain := "osmosis-1"
// node := "https://rpc.osmosis.zone:443"
// osmosisHomeDir := "/home/kyle/.osmosisd"
//
//	keyringBackend := "test"
func GetOsmosisTxClient(encodingConfig osmosisParams.EncodingConfig, chain string, node string, osmosisHomeDir string, keyringBackend string, fromFlag string) (client.Context, error) {
	//encodingConfig := osmosis.MakeEncodingConfig()
	clientCtx := client.Context{
		ChainID:      chain,
		NodeURI:      node,
		KeyringDir:   osmosisHomeDir,
		GenerateOnly: false,
	}

	ctxKeyring, krErr := client.NewKeyringFromBackend(clientCtx, keyringBackend)
	if krErr != nil {
		return clientCtx, krErr
	}

	clientCtx = clientCtx.WithKeyring(ctxKeyring)

	//Where node is the node RPC URI
	rpcClient, rpcErr := client.NewClientFromNode(node)

	if rpcErr != nil {
		return clientCtx, rpcErr
	}

	fromAddr, fromName, _, err := client.GetFromFields(clientCtx.Keyring, fromFlag, clientCtx.GenerateOnly)
	if err != nil {
		return clientCtx, err
	}

	clientCtx = clientCtx.WithCodec(encodingConfig.Marshaler).
		WithChainID(chain).
		WithFrom(fromFlag).
		WithFromAddress(fromAddr).
		WithFromName(fromName).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authTypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastAsync).
		WithHomeDir(osmosisHomeDir).
		WithViper("OSMOSIS").
		WithNodeURI(node).
		WithClient(rpcClient).
		WithSkipConfirmation(true)

	return clientCtx, nil
}
