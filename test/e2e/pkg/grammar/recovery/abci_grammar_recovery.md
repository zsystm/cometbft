```
package "github.com/cometbft/cometbft/test/e2e/pkg/grammar/recovery/grammar-auto"

Start : Recovery ; 

Recovery :  ConsensusExec ;

ConsensusExec : ConsensusHeights ;
ConsensusHeights : ConsensusHeight | ConsensusHeight ConsensusHeights ;
ConsensusHeight : ConsensusRounds FinalizeBlock Commit | FinalizeBlock Commit ;
ConsensusRounds : ConsensusRound | ConsensusRound ConsensusRounds ;
ConsensusRound : Proposer | NonProposer ; 

Proposer : GotVotes | ProposerSimple | Extend | GotVotes ProposerSimple | GotVotes Extend | ProposerSimple Extend | GotVotes ProposerSimple Extend ; 
ProposerSimple : PrepareProposal | PrepareProposal ProcessProposal ;
NonProposer: GotVotes | ProcessProposal | Extend | GotVotes ProcessProposal | GotVotes Extend | ProcessProposal Extend | GotVotes ProcessProposal Extend ; 
Extend : ExtendVote | GotVotes ExtendVote | ExtendVote GotVotes | GotVotes ExtendVote GotVotes ;
GotVotes : GotVote | GotVote GotVotes ; 

FinalizeBlock : "finalize_block" ; 
Commit : "commit" ;
PrepareProposal : "prepare_proposal" ; 
ProcessProposal : "process_proposal" ;
ExtendVote : "extend_vote" ;
GotVote : "verify_vote_extension" ;

```

The part of the original grammar (https://github.com/cometbft/cometbft/blob/main/spec/abci/abci%2B%2B_comet_expected_behavior.md) the grammar above 
refers to is below: 

start               = recovery

recovery            = info consensus-exec

consensus-exec      = (inf)consensus-height
consensus-height    = *consensus-round finalize-block commit
consensus-round     = proposer / non-proposer

proposer            = *got-vote [prepare-proposal [process-proposal]] [extend]
extend              = *got-vote extend-vote *got-vote
non-proposer        = *got-vote [process-proposal] [extend]

info                = %s"<Info>"
prepare-proposal    = %s"<PrepareProposal>"
process-proposal    = %s"<ProcessProposal>"
extend-vote         = %s"<ExtendVote>"
got-vote            = %s"<VerifyVoteExtension>"
finalize-block      = %s"<FinalizeBlock>"
commit              = %s"<Commit>"

