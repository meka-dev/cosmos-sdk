package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	sdk_simapp "cosmossdk.io/simapp"
	sdk_simapp_params "cosmossdk.io/simapp/params"
	sdk_x_builder_types "cosmossdk.io/x/mev/types"
	sdk_std "github.com/cosmos/cosmos-sdk/std"
)

var encodingConfig = func() sdk_simapp_params.EncodingConfig {
	encodingConfig := sdk_simapp_params.MakeTestEncodingConfig()
	sdk_std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	sdk_std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	sdk_simapp.ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	sdk_simapp.ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}()

func main() {
	log.SetFlags(0)

	fs := flag.NewFlagSet("turbobuilder", flag.ContinueOnError)
	var (
		listenAddr       = fs.String("listen-addr", "localhost:9099", "listen address for builder API")
		builderKeyFile   = fs.String("builder-key-file", "", "builder private key file (simd query builder export-json-key)")
		nodeURI          = fs.String("node-uri", "http://localhost:26657", "Tendermint RPC URI of trusted node")
		disableNodeCache = fs.Bool("disable-node-cache", false, "disable local cache of metadata from trusted node")
	)
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	var builderKey *sdk_x_builder_types.Key
	{
		key, err := sdk_x_builder_types.LoadKey(*builderKeyFile, encodingConfig.Codec)
		if err != nil {
			log.Fatalf("invalid privkey file: %s", err)
		}

		builderKey = key
	}

	log.Printf("builder address %s", builderKey.Address)

	var directNode *DirectNode
	{
		n, err := NewDirectNode(*nodeURI)
		if err != nil {
			log.Fatal(err)
		}

		directNode = n
	}

	var node Node
	{
		switch *disableNodeCache {
		case true:
			node = directNode

		case false:
			nodeWithCache := &NodeWithCache{DirectNode: directNode}

			go func() {
				log := log.New(log.Writer(), "RefreshProposers: ", log.Flags())
				for range time.Tick(time.Second) {
					if err := nodeWithCache.RefreshProposers(ctx); err != nil {
						log.Printf("error: %v", err)
					}
				}
			}()

			go func() {
				log := log.New(log.Writer(), "RefreshAccounts: ", log.Flags())
				for range time.Tick(time.Second) {
					if err := nodeWithCache.RefreshAccounts(ctx, builderKey.Address); err != nil {
						log.Printf("error: %v", err)
					}
				}
			}()

			log.Printf("waiting for node cache to be ready...")
			if err := nodeWithCache.WaitForReady(ctx); err != nil {
				log.Fatalf("node cache didn't become ready: %v", err)
			}

			log.Printf("node cache is ready")

			node = nodeWithCache
		}
	}

	var store Store
	{
		// The store is how we persist bids, including tracking state over time,
		// such as commitments from proposers.
		store = NewMemStore()

		// Regularly clean old bids out of the store.
		go func() {
			for range time.Tick(10 * time.Second) {
				if err := CleanStore(ctx, store); err != nil {
					log.Printf("error cleaning store: %v", err)
				}
			}
		}()

		// Regularly check the store for winning bids, and verify they are
		// included in the auction block, including submitting evidence when
		// necessary.
		go func() {
			for range time.Tick(3 * time.Second) {
				if err := VerifyBids(ctx, store, node, builderKey); err != nil {
					log.Printf("error verifying bids: %v", err)
				}
			}
		}()
	}

	var api *API
	{
		a, err := NewAPI(ctx, builderKey, node, store)
		if err != nil {
			log.Fatal(err)
		}

		api = a
	}

	var handler http.Handler
	{
		handler = api
		handler = &loggingMiddleware{Handler: handler, logger: log.New(log.Writer(), "API: ", log.Flags())}
	}

	log.Printf("builder API listening on %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, handler))
}
