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

package verification

import (
	"github.com/berachain/beacon-kit/mod/errors"
	"github.com/berachain/beacon-kit/mod/log"
	"github.com/berachain/beacon-kit/mod/primitives"
	engineprimitives "github.com/berachain/beacon-kit/mod/primitives-engine"
	"github.com/berachain/beacon-kit/mod/state-transition/pkg/core/state"
)

// PayloadVerifier is responsible for verifying incoming execution
// payloads to ensure they are valid.
type PayloadVerifier struct {
	cs     primitives.ChainSpec
	logger log.Logger[any]
}

// NewPayloadVerifier creates a new payload validator.
func NewPayloadVerifier(
	cs primitives.ChainSpec, logger log.Logger[any],
) *PayloadVerifier {
	return &PayloadVerifier{
		cs:     cs,
		logger: logger,
	}
}

// VerifyPayload verifies the incoming payload.
func (pv *PayloadVerifier) VerifyPayload(
	st state.BeaconState,
	payload engineprimitives.ExecutionPayload,
) error {
	latestExecutionPayloadHeader, err := st.GetLatestExecutionPayloadHeader()
	if err != nil {
		return err
	}
	safeHash := latestExecutionPayloadHeader.GetBlockHash()

	if safeHash != payload.GetParentHash() {
		return errors.Newf(
			"parent block with hash %x is not finalized, expected finalized hash %x",
			payload.GetParentHash(),
			safeHash,
		)
	}

	// Get the current epoch.
	slot, err := st.GetSlot()
	if err != nil {
		return err
	}

	// When we are verifying a payload we expect that it was produced by
	// the proposer for the slot that it is for.
	expectedMix, err := st.GetRandaoMixAtIndex(
		uint64(pv.cs.SlotToEpoch(slot)) % pv.cs.EpochsPerHistoricalVector())
	if err != nil {
		return err
	}

	// Ensure the prev randao matches the local state.
	if payload.GetPrevRandao() != expectedMix {
		return errors.Newf(
			"prev randao does not match, expected: %x, got: %x",
			expectedMix, payload.GetPrevRandao(),
		)
	}

	// TODO: Verify timestamp data once Clock is done.
	// if expectedTime, err := spec.TimeAtSlot(slot, genesisTime); err != nil {
	// 	return errors.Newf("slot or genesis time in state is corrupt, cannot
	// compute time: %v", err)
	// } else if payload.Timestamp != expectedTime {
	// 	return errors.Newf("state at slot %d, genesis time %d, expected
	// execution
	// payload time %d, but got %d",
	// 		slot, genesisTime, expectedTime, payload.Timestamp)
	// }

	// Verify the number of withdrawals.
	if withdrawals := payload.GetWithdrawals(); uint64(
		len(payload.GetWithdrawals()),
	) > pv.cs.MaxWithdrawalsPerPayload() {
		return errors.Newf(
			"too many withdrawals, expected: %d, got: %d",
			pv.cs.MaxWithdrawalsPerPayload(), len(withdrawals),
		)
	}

	return nil
}