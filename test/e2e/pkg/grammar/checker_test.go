package grammar

import (
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"
)

var (
	initChain       = &abci.Request{Value: &abci.Request_InitChain{InitChain: &abci.RequestInitChain{}}}
	finalizeBlock   = &abci.Request{Value: &abci.Request_FinalizeBlock{FinalizeBlock: &abci.RequestFinalizeBlock{}}}
	commit          = &abci.Request{Value: &abci.Request_Commit{Commit: &abci.RequestCommit{}}}
	offerSnapshot   = &abci.Request{Value: &abci.Request_OfferSnapshot{OfferSnapshot: &abci.RequestOfferSnapshot{}}}
	applyChunk      = &abci.Request{Value: &abci.Request_ApplySnapshotChunk{ApplySnapshotChunk: &abci.RequestApplySnapshotChunk{}}}
	prepareProposal = &abci.Request{Value: &abci.Request_PrepareProposal{PrepareProposal: &abci.RequestPrepareProposal{}}}
	processProposal = &abci.Request{Value: &abci.Request_ProcessProposal{ProcessProposal: &abci.RequestProcessProposal{}}}
	extendVote      = &abci.Request{Value: &abci.Request_ExtendVote{ExtendVote: &abci.RequestExtendVote{}}}
	gotVote         = &abci.Request{Value: &abci.Request_VerifyVoteExtension{VerifyVoteExtension: &abci.RequestVerifyVoteExtension{}}}
)

const (
	CleanStart = true
	Pass       = true
	Fail       = false
)

func TestVerify(t *testing.T) {
	tests := []struct {
		name         string
		abciCalls    []*abci.Request
		isCleanStart bool
		result       bool
	}{
		// start = clean-start
		// clean-start = init-chain consensus-exec
		// consensus-height = finalizeBlock commit
		{"empty-block-1", []*abci.Request{initChain, finalizeBlock, commit}, CleanStart, Pass},
		{"consensus-exec-missing", []*abci.Request{initChain}, CleanStart, Fail},
		{"finalize-block-missing-1", []*abci.Request{initChain, commit}, CleanStart, Fail},
		{"commit-missing-1", []*abci.Request{initChain, finalizeBlock}, CleanStart, Fail},
		// consensus-height = *consensus-round finalizeBlock commit
		// consensus-round = proposer
		// proposer = *gotVote
		{"proposer-round-1", []*abci.Request{initChain, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-2", []*abci.Request{initChain, gotVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		// proposer = [prepare-proposal [process-proposal]]
		{"proposer-round-3", []*abci.Request{initChain, prepareProposal, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-4", []*abci.Request{initChain, prepareProposal, finalizeBlock, commit}, CleanStart, Pass},
		// proposer = [extend]
		{"proposer-round-5", []*abci.Request{initChain, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-6", []*abci.Request{initChain, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-7", []*abci.Request{initChain, gotVote, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-8", []*abci.Request{initChain, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-9", []*abci.Request{initChain, extendVote, gotVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-10", []*abci.Request{initChain, gotVote, gotVote, extendVote, gotVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		// proposer = *gotVote [prepare-proposal [process-proposal]]
		{"proposer-round-11", []*abci.Request{initChain, gotVote, prepareProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-12", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-13", []*abci.Request{initChain, gotVote, prepareProposal, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-14", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		// proposer = *gotVote [extend]
		// same as just [extend]

		// proposer = [prepare-proposal [process-proposal]] [extend]
		{"proposer-round-15", []*abci.Request{initChain, prepareProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-16", []*abci.Request{initChain, prepareProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-17", []*abci.Request{initChain, prepareProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-18", []*abci.Request{initChain, prepareProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-19", []*abci.Request{initChain, prepareProposal, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-20", []*abci.Request{initChain, prepareProposal, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-21", []*abci.Request{initChain, prepareProposal, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-22", []*abci.Request{initChain, prepareProposal, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		// proposer = *gotVote [prepare-proposal [process-proposal]] [extend]
		{"proposer-round-23", []*abci.Request{initChain, gotVote, prepareProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-24", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-25", []*abci.Request{initChain, gotVote, prepareProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-26", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-27", []*abci.Request{initChain, gotVote, prepareProposal, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-28", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-29", []*abci.Request{initChain, gotVote, prepareProposal, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"proposer-round-30", []*abci.Request{initChain, gotVote, gotVote, prepareProposal, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},

		// consensus-round = non-proposer
		// non-proposer = *gotVote
		// same as for proposer

		// non-proposer = [process-proposal]
		{"non-proposer-round-1", []*abci.Request{initChain, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		// non-proposer = [extend]
		// same as for proposer

		// non-proposer = *gotVote [process-proposal]
		{"non-proposer-round-2", []*abci.Request{initChain, gotVote, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-3", []*abci.Request{initChain, gotVote, gotVote, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		// non-proposer = *gotVote [extend]
		// same as just [extend]

		// non-proposer = [process-proposal] [extend]
		{"non-proposer-round-4", []*abci.Request{initChain, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-5", []*abci.Request{initChain, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-6", []*abci.Request{initChain, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-7", []*abci.Request{initChain, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},

		// non-proposer = *gotVote [prepare-proposal [process-proposal]] [extend]
		{"non-proposer-round-8", []*abci.Request{initChain, gotVote, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-9", []*abci.Request{initChain, gotVote, gotVote, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-10", []*abci.Request{initChain, gotVote, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"non-proposer-round-11", []*abci.Request{initChain, gotVote, gotVote, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},

		{"multiple-rounds-1", []*abci.Request{initChain, prepareProposal, processProposal, processProposal, prepareProposal, processProposal, processProposal, processProposal, finalizeBlock, commit}, CleanStart, Pass},

		// clean-start = init-chain state-sync consensus-exec
		// state-sync = success-sync
		{"one-apply-chunk-1", []*abci.Request{initChain, offerSnapshot, applyChunk, finalizeBlock, commit}, CleanStart, Pass},
		{"multiple-apply-chunks-1", []*abci.Request{initChain, offerSnapshot, applyChunk, applyChunk, finalizeBlock, commit}, CleanStart, Pass},
		{"offer-snapshot-missing-1", []*abci.Request{initChain, applyChunk, finalizeBlock, commit}, CleanStart, Fail},
		{"apply-chunk-missing", []*abci.Request{initChain, offerSnapshot, finalizeBlock, commit}, CleanStart, Fail},
		// state-sync = *state-sync-attempt success-sync
		{"one-apply-chunk-2", []*abci.Request{initChain, offerSnapshot, applyChunk, offerSnapshot, applyChunk, finalizeBlock, commit}, CleanStart, Pass},
		{"multiple-apply-chunks-2", []*abci.Request{initChain, offerSnapshot, applyChunk, applyChunk, applyChunk, offerSnapshot, applyChunk, finalizeBlock, commit}, CleanStart, Pass},
		{"offer-snapshot-missing-2", []*abci.Request{initChain, applyChunk, offerSnapshot, applyChunk, finalizeBlock, commit}, CleanStart, Fail},
		{"no-apply-chunk", []*abci.Request{initChain, offerSnapshot, offerSnapshot, applyChunk, finalizeBlock, commit}, CleanStart, Pass},

		// start = recovery
		// recovery = consensus-exec
		// consensus-height = finalizeBlock commit
		{"empty-block-2", []*abci.Request{finalizeBlock, commit}, !CleanStart, Pass},
		{"finalize-block-missing-2", []*abci.Request{commit}, !CleanStart, Fail},
		{"commit-missing-2", []*abci.Request{finalizeBlock}, !CleanStart, Fail},
		// consensus-height = *consensus-round finalizeBlock commit
		// consensus-round = proposer
		// proposer = *gotVote
		{"rec-proposer-round-1", []*abci.Request{gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-2", []*abci.Request{gotVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		// proposer = [prepare-proposal [process-proposal]]
		{"rec-proposer-round-3", []*abci.Request{prepareProposal, processProposal, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-4", []*abci.Request{prepareProposal, finalizeBlock, commit}, !CleanStart, Pass},
		// proposer = [extend]
		{"rec-proposer-round-5", []*abci.Request{extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-6", []*abci.Request{gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-7", []*abci.Request{gotVote, gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-8", []*abci.Request{extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-9", []*abci.Request{extendVote, gotVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-10", []*abci.Request{gotVote, gotVote, extendVote, gotVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		// proposer = *gotVote [prepare-proposal [process-proposal]]
		{"rec-proposer-round-11", []*abci.Request{gotVote, prepareProposal, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-12", []*abci.Request{gotVote, gotVote, prepareProposal, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-13", []*abci.Request{gotVote, prepareProposal, processProposal, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-14", []*abci.Request{gotVote, gotVote, prepareProposal, processProposal, finalizeBlock, commit}, !CleanStart, Pass},
		// proposer = *gotVote [extend]
		// same as just [extend]

		// proposer = [prepare-proposal [process-proposal]] [extend]
		{"rec-proposer-round-15", []*abci.Request{prepareProposal, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-16", []*abci.Request{prepareProposal, gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-17", []*abci.Request{prepareProposal, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-18", []*abci.Request{prepareProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-19", []*abci.Request{prepareProposal, processProposal, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-20", []*abci.Request{prepareProposal, processProposal, gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-21", []*abci.Request{prepareProposal, processProposal, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-22", []*abci.Request{prepareProposal, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		// proposer = *gotVote [prepare-proposal [process-proposal]] [extend]
		{"rec-proposer-round-23", []*abci.Request{gotVote, prepareProposal, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-24", []*abci.Request{gotVote, gotVote, prepareProposal, gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-25", []*abci.Request{gotVote, prepareProposal, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-26", []*abci.Request{gotVote, gotVote, prepareProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-27", []*abci.Request{gotVote, prepareProposal, processProposal, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-28", []*abci.Request{gotVote, gotVote, prepareProposal, processProposal, gotVote, extendVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-29", []*abci.Request{gotVote, prepareProposal, processProposal, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},
		{"rec-proposer-round-30", []*abci.Request{gotVote, gotVote, prepareProposal, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, !CleanStart, Pass},

		// consensus-round = non-proposer
		// non-proposer = *gotVote
		// same as for proposer

		// non-proposer = [process-proposal]
		{"rec-non-proposer-round-1", []*abci.Request{initChain, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		// non-proposer = [extend]
		// same as for proposer

		// non-proposer = *gotVote [process-proposal]
		{"rec-non-proposer-round-2", []*abci.Request{initChain, gotVote, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-3", []*abci.Request{initChain, gotVote, gotVote, processProposal, finalizeBlock, commit}, CleanStart, Pass},
		// non-proposer = *gotVote [extend]
		// same as just [extend]

		// non-proposer = [process-proposal] [extend]
		{"rec-non-proposer-round-4", []*abci.Request{initChain, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-5", []*abci.Request{initChain, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-6", []*abci.Request{initChain, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-7", []*abci.Request{initChain, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},

		// non-proposer = *gotVote [prepare-proposal [process-proposal]] [extend]
		{"rec-non-proposer-round-8", []*abci.Request{initChain, gotVote, processProposal, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-9", []*abci.Request{initChain, gotVote, gotVote, processProposal, gotVote, extendVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-10", []*abci.Request{initChain, gotVote, processProposal, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"rec-non-proposer-round-11", []*abci.Request{initChain, gotVote, gotVote, processProposal, gotVote, extendVote, gotVote, finalizeBlock, commit}, CleanStart, Pass},
		{"multiple-rounds-2", []*abci.Request{prepareProposal, processProposal, processProposal, prepareProposal, processProposal, processProposal, processProposal, finalizeBlock, commit}, !CleanStart, Pass},

		// corner cases
		{"empty execution", nil, CleanStart, Fail},
		{"empty execution", nil, !CleanStart, Fail},
	}

	for _, test := range tests {
		checker := NewGrammarChecker(DefaultConfig())
		result, err := checker.Verify(test.abciCalls, test.isCleanStart)
		if result == test.result {
			continue
		}
		if err == nil {
			err = fmt.Errorf("grammar parsed an incorrect execution: %v", checker.getExecutionString(test.abciCalls))
		}
		t.Errorf("Test %v returned %v, expected %v\n%v\n", test.name, result, test.result, err)
	}
}

func TestFilterLastHeight(t *testing.T) {
	reqs := []*abci.Request{initChain, finalizeBlock, commit}
	checker := NewGrammarChecker(DefaultConfig())
	rr, n := checker.filterLastHeight(reqs)
	require.Equal(t, len(reqs), len(rr))
	require.Zero(t, n)

	reqs = append(reqs, finalizeBlock)
	rrr, n := checker.filterLastHeight(reqs)
	require.Equal(t, len(rr), len(rrr))
	require.Equal(t, n, 1)
}
