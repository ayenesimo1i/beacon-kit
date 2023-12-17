// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2023, Berachain Foundation. All rights reserved.
// Use of this software is govered by the Business Source License included
// in the LICENSE file of this repository and at www.mariadb.com/bsl11.
//
// ANY USE OF THE LICENSED WORK IN VIOLATION OF THIS LICENSE WILL AUTOMATICALLY
// TERMINATE YOUR RIGHTS UNDER THIS LICENSE FOR THE CURRENT AND ALL OTHER
// VERSIONS OF THE LICENSED WORK.
//
// THIS LICENSE DOES NOT GRANT YOU ANY RIGHT IN ANY TRADEMARK OR LOGO OF
// LICENSOR OR ITS AFFILIATES (PROVIDED THAT YOU MAY USE A TRADEMARK OR LOGO OF
// LICENSOR AS EXPRESSLY REQUIRED BY THIS LICENSE).
//
// TO THE EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSED WORK IS PROVIDED ON
// AN “AS IS” BASIS. LICENSOR HEREBY DISCLAIMS ALL WARRANTIES AND CONDITIONS,
// EXPRESS OR IMPLIED, INCLUDING (WITHOUT LIMITATION) WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, NON-INFRINGEMENT, AND
// TITLE.

package runtime

import (
	"context"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/itsdevbear/bolaris/beacon/execution"
	authethclient "github.com/itsdevbear/bolaris/beacon/execution/eth"
	proposal "github.com/itsdevbear/bolaris/cosmos/abci/proposal"
	"github.com/itsdevbear/bolaris/cosmos/config"
	"github.com/itsdevbear/bolaris/cosmos/runtime/miner"
	evmkeeper "github.com/itsdevbear/bolaris/cosmos/x/evm/keeper"
)

// EVMKeeper is an interface that defines the methods needed for the EVM setup.
type EVMKeeper interface {
	// Setup initializes the EVM keeper.
	Setup(*execution.Service) error
}

// CosmosApp is an interface that defines the methods needed for the Cosmos setup.
type CosmosApp interface {
	SetPrepareProposal(sdk.PrepareProposalHandler)
	baseapp.ProposalTxVerifier
	SetMempool(mempool.Mempool)
	SetAnteHandler(sdk.AnteHandler)
	SetExtendVoteHandler(sdk.ExtendVoteHandler)
	SetProcessProposal(sdk.ProcessProposalHandler)
	SetVerifyVoteExtensionHandler(sdk.VerifyVoteExtensionHandler)
	ChainID() string
}

// Polaris is a struct that wraps the Polaris struct from the polar package.
type Polaris struct {
	*execution.Service
	// WrappedMiner is a wrapped version of the Miner component.
	WrappedMiner *miner.Miner
	// logger is the underlying logger supplied by the sdk.
	logger log.Logger
}

// New creates a new Polaris runtime from the provided
// dependencies.
func New(
	appOpts servertypes.AppOptions,
	logger log.Logger,
) (*Polaris, error) {
	var err error
	p := &Polaris{
		logger: logger,
	}

	// Read the configuration from the cosmos app options
	cfg, err := config.ReadConfigFromAppOpts(appOpts)
	if err != nil {
		return nil, err
	}
	// Connect to the execution client.
	var ethClient *ethclient.Client
	ethClient, err = authethclient.NewAuthenticatedEthClient(
		cfg.ExecutionClient.RPCDialURL, cfg.ExecutionClient.JWTSecretPath, logger,
	)

	p.Service = execution.NewEngineClientService(ethClient)

	if err != nil {
		return nil, err
	}

	return p, nil
}

// New creates a new Polaris runtime from the provided
// dependencies, panics on error.
func MustNew(appOpts servertypes.AppOptions, logger log.Logger) *Polaris {
	p, err := New(appOpts, logger)
	if err != nil {
		panic(err)
	}
	return p
}

// Build is a function that sets up the Polaris struct.
// It takes a BaseApp and an EVMKeeper as arguments.
// It returns an error if the setup fails.
func (p *Polaris) Build(app CosmosApp, vs baseapp.ValidatorStore, ek *evmkeeper.Keeper) error {
	// todo use `vs` later?
	_ = vs
	mempool := mempool.NewSenderNonceMempool()
	app.SetMempool(mempool)
	p.WrappedMiner = miner.New(p.Service, ek, p.logger)

	// Create the proposal handler that will be used to fill proposals with
	// transactions and oracle data.
	// proposalHandler := proposal.NewProposalHandler(
	// 	p.logger,
	// 	baseapp.NoOpPrepareProposal(),
	// 	baseapp.NoOpProcessProposal(),
	// 	ve.NewDefaultValidateVoteExtensionsFn(app.ChainID(), vs),
	// 	ve.NewProcessor(p.WrappedMiner, ek, p.logger).ProcessCommitInfo,
	// )

	defaultProposalHandler := baseapp.NewDefaultProposalHandler(mempool, app)
	proposalHandler := proposal.NewProposalHandler2(p.WrappedMiner,
		defaultProposalHandler.PrepareProposalHandler(), defaultProposalHandler.ProcessProposalHandler())
	app.SetPrepareProposal(proposalHandler.PrepareProposalHandler)
	app.SetProcessProposal(proposalHandler.ProcessProposalHandler)

	// if err := p.WrappedMiner.SyncEl(context.Background()); err != nil {
	// 	return err
	// }

	// // Create the vote extensions handler that will be used to extend and verify
	// // vote extensions (i.e. oracle data).
	// voteExtensionsHandler := ve.NewVoteExtensionHandler(
	// 	p.logger,
	// 	time.Second,
	// 	p.WrappedMiner,
	// )
	// app.SetExtendVoteHandler(voteExtensionsHandler.ExtendVoteHandler())
	// app.SetVerifyVoteExtensionHandler(voteExtensionsHandler.VerifyVoteExtensionHandler())

	return nil
}

func (p *Polaris) SyncEL(ctx context.Context) error {
	return p.WrappedMiner.SyncEl(ctx)
}
