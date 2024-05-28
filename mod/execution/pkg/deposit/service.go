// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package deposit

import (
	"context"

	"github.com/berachain/beacon-kit/mod/log"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
)

// Service represenst the deposit service that processes deposit events.
type Service[
	BeaconBlockT BeaconBlock,
	BlockEventT BlockEvent[BeaconBlockT],
	DepositStoreT Store[DepositT],
	SubscriptionT interface {
		Unsubscribe()
	},
	DepositT any,
] struct {
	// feed is the block feed that provides block events.
	feed BlockFeed[BeaconBlockT, BlockEventT, SubscriptionT]
	// dc is the contract interface for interacting with the deposit contract.
	dc Contract[DepositT]
	// sb is the storage backend for managing deposit operations.
	sb StorageBackend[
		any, any, any, DepositStoreT, DepositT,
	]
	// eth1FollowDistance is the follow distance for Ethereum 1.0 blocks.
	eth1FollowDistance math.U64
	// logger is used for logging information and errors.
	logger log.Logger[any]
}

// NewService creates a new instance of the Service struct.
func NewService[
	BeaconBlockT BeaconBlock,
	BlockEventT BlockEvent[BeaconBlockT],
	DepositStoreT Store[DepositT],
	SubscriptionT interface {
		Unsubscribe()
	},
	DepositT any,
](
	feed BlockFeed[BeaconBlockT, BlockEventT, SubscriptionT],
	logger log.Logger[any],
	sb StorageBackend[
		any, any, any, DepositStoreT, DepositT,
	],
	dc Contract[DepositT],
) *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
] {
	return &Service[
		BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
	]{
		feed:   feed,
		logger: logger,
		sb:     sb,
		dc:     dc,
		// eth1FollowDistance is set to 1 by default. This value should be
		// configurable.
		eth1FollowDistance: 1,
	}
}

// Start starts the service and begins processing block events.
func (s *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
]) Start(
	ctx context.Context,
) error {
	ch := make(chan BlockEventT)
	sub := s.feed.Subscribe(ch)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-ch:
				if err := s.handleDepositEvent(event); err != nil {
					s.logger.Error("failed to handle deposit event", "err", err)
				}
			}
		}
	}()

	return nil
}

// Name returns the name of the service.
func (s *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
]) Name() string {
	return "deposit-handler"
}

// Status returns the current status of the service.
func (s *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
]) Status() error {
	return nil
}

// WaitForHealthy waits for the service to become healthy.
func (s *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
]) WaitForHealthy(
	_ context.Context,
) {
}

// handleDepositEvent processes a deposit event.
func (s *Service[
	BeaconBlockT, BlockEventT, DepositStoreT, SubscriptionT, DepositT,
]) handleDepositEvent(
	e BlockEventT,
) error {
	// slot is the block slot number adjusted by the follow distance.
	slot := e.Block().GetSlot() - s.eth1FollowDistance
	s.logger.Info("💵 processing deposit logs 💵", "slot", slot)
	// deposits are retrieved from the deposit contract.
	deposits, err := s.dc.GetDeposits(e.Context(), slot.Unwrap())
	if err != nil {
		return err
	}

	// Enqueue the deposits into the deposit store.
	if err = s.sb.DepositStore(e.Context()).EnqueueDeposits(
		deposits,
	); err != nil {
		return err
	}
	return nil
}