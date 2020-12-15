package group

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/gogo/protobuf/types"
	"github.com/regen-network/regen-ledger/math"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThresholdDecisionPolicy(t *testing.T) {
	specs := map[string]struct {
		srcPolicy         ThresholdDecisionPolicy
		srcTally          Tally
		srcTotalPower     string
		srcVotingDuration time.Duration
		expResult         DecisionPolicyResult
		expErr            error
	}{
		"accept when yes count greater than threshold": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "2"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: true, Final: true},
		},
		"accept when yes count equal to threshold": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "1", NoCount: "0", AbstainCount: "0", VetoCount: "0"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: true, Final: true},
		},
		"reject when yes count lower to threshold": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "0", NoCount: "0", AbstainCount: "0", VetoCount: "0"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: false, Final: false},
		},
		"reject as final when remaining votes can't cross threshold": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "2",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "0", NoCount: "2", AbstainCount: "0", VetoCount: "0"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: false, Final: true},
		},
		"expired when on timeout": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "2"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Second,
			expResult:         DecisionPolicyResult{Allow: false, Final: true},
		},
		"expired when after timeout": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "2"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Second + time.Nanosecond,
			expResult:         DecisionPolicyResult{Allow: false, Final: true},
		},
		"abstain has no impact": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "0", NoCount: "0", AbstainCount: "1", VetoCount: "0"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: false, Final: false},
		},
		"veto same as no": {
			srcPolicy: ThresholdDecisionPolicy{
				Threshold: "1",
				Timeout:   proto.Duration{Seconds: 1},
			},
			srcTally:          Tally{YesCount: "0", NoCount: "0", AbstainCount: "0", VetoCount: "2"},
			srcTotalPower:     "3",
			srcVotingDuration: time.Millisecond,
			expResult:         DecisionPolicyResult{Allow: false, Final: false},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			res, err := spec.srcPolicy.Allow(spec.srcTally, spec.srcTotalPower, spec.srcVotingDuration)
			if spec.expErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, spec.expResult, res)
		})
	}
}

func TestThresholdDecisionPolicyValidate(t *testing.T) {
	specs := map[string]struct {
		src    ThresholdDecisionPolicy
		expErr bool
	}{
		"all good": {src: ThresholdDecisionPolicy{
			Threshold: "1",
			Timeout:   proto.Duration{Seconds: 1},
		}},
		"greater than group total weight": {
			src: ThresholdDecisionPolicy{
				Threshold: "2",
				Timeout:   proto.Duration{Seconds: 1},
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.Validate(GroupInfo{TotalWeight: "1"})
			assert.Equal(t, spec.expErr, err != nil, err)
		})
	}
}

func TestThresholdDecisionPolicyValidateBasic(t *testing.T) {
	maxSeconds := int64(10000 * 365.25 * 24 * 60 * 60)
	specs := map[string]struct {
		src    ThresholdDecisionPolicy
		expErr bool
	}{
		"all good": {src: ThresholdDecisionPolicy{
			Threshold: "1",
			Timeout:   proto.Duration{Seconds: 1},
		}},
		"threshold missing": {src: ThresholdDecisionPolicy{
			Timeout: proto.Duration{Seconds: 1},
		},
			expErr: true,
		},
		"timeout missing": {src: ThresholdDecisionPolicy{
			Threshold: "1",
		},
			expErr: true,
		},
		"duration out of limit": {src: ThresholdDecisionPolicy{
			Threshold: "1",
			Timeout:   proto.Duration{Seconds: maxSeconds + 1},
		},
			expErr: true,
		},
		"no negative thresholds": {src: ThresholdDecisionPolicy{
			Threshold: "-1",
			Timeout:   proto.Duration{Seconds: 1},
		},
			expErr: true,
		},
		"no empty thresholds": {src: ThresholdDecisionPolicy{
			Timeout: proto.Duration{Seconds: 1},
		},
			expErr: true,
		},
		"no zero thresholds": {src: ThresholdDecisionPolicy{
			Timeout:   proto.Duration{Seconds: 1},
			Threshold: "0",
		},
			expErr: true,
		},
		"no negative timeouts": {src: ThresholdDecisionPolicy{
			Threshold: "1",
			Timeout:   proto.Duration{Seconds: -1},
		},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			assert.Equal(t, spec.expErr, err != nil, err)
		})
	}
}

func TestVoteNaturalKey(t *testing.T) {
	v := Vote{
		ProposalId: 1,
		Voter:      []byte{0xff, 0xfe},
	}
	assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 1, 0xff, 0xfe}, v.NaturalKey())
}

func TestGroupInfoValidation(t *testing.T) {
	specs := map[string]struct {
		src    GroupInfo
		expErr bool
	}{
		"all good": {
			src: GroupInfo{
				GroupId:     1,
				Admin:       []byte("valid--admin-address"),
				Comment:     "any",
				Version:     1,
				TotalWeight: "0",
			},
		},
		"invalid group": {
			src: GroupInfo{
				Admin:       []byte("valid--admin-address"),
				Comment:     "any",
				Version:     1,
				TotalWeight: "0",
			},
			expErr: true,
		},
		"invalid admin": {
			src: GroupInfo{
				GroupId:     1,
				Admin:       []byte(""),
				Comment:     "any",
				Version:     1,
				TotalWeight: "0",
			},
			expErr: true,
		},
		"invalid version": {
			src: GroupInfo{
				GroupId:     1,
				Admin:       []byte("valid--admin-address"),
				Comment:     "any",
				TotalWeight: "0",
			},
			expErr: true,
		},
		"unset total weight": {
			src: GroupInfo{
				GroupId: 1,
				Admin:   []byte("valid--admin-address"),
				Comment: "any",
				Version: 1,
			},
			expErr: true,
		},
		"negative total weight": {
			src: GroupInfo{
				GroupId:     1,
				Admin:       []byte("valid--admin-address"),
				Comment:     "any",
				Version:     1,
				TotalWeight: "-1",
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGroupMemberValidation(t *testing.T) {
	specs := map[string]struct {
		src    GroupMember
		expErr bool
	}{
		"all good": {
			src: GroupMember{
				GroupId: 1,
				Member:  []byte("valid-member-address"),
				Weight:  "1",
				Comment: "any",
			},
		},
		"invalid group": {
			src: GroupMember{
				GroupId: 0,
				Member:  []byte("valid-member-address"),
				Weight:  "1",
				Comment: "any",
			},
			expErr: true,
		},
		"invalid address": {
			src: GroupMember{
				GroupId: 1,
				Member:  []byte("invalid-member-address"),
				Weight:  "1",
				Comment: "any",
			},
			expErr: true,
		},
		"empy address": {
			src: GroupMember{
				GroupId: 1,
				Weight:  "1",
				Comment: "any",
			},
			expErr: true,
		},
		"invalid weight": {
			src: GroupMember{
				GroupId: 1,
				Member:  []byte("valid-member-address"),
				Weight:  "0",
				Comment: "any",
			},
			expErr: true,
		},
		"nil weight": {
			src: GroupMember{
				GroupId: 1,
				Member:  []byte("valid-member-address"),
				Comment: "any",
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGroupAccountInfo(t *testing.T) {
	specs := map[string]struct {
		groupAccount sdk.AccAddress
		group        ID
		admin        sdk.AccAddress
		comment      string
		version      uint64
		threshold    string
		timeout      proto.Duration
		expErr       bool
	}{
		"all good": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
		},
		"invalid group": {
			group:        0,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"invalid group account address": {
			group:        1,
			groupAccount: []byte("any-invalid-group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"empty group account address": {
			group:     1,
			admin:     []byte("valid--admin-address"),
			comment:   "any",
			version:   1,
			threshold: "1",
			timeout:   proto.Duration{Seconds: 1},
			expErr:    true,
		},
		"empty admin account address": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"invalid admin account address": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("any-invalid-admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"empty version number": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			threshold:    "1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"missing decision policy": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			expErr:       true,
		},
		"missing decision policy timeout": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			expErr:       true,
		},
		"decision policy with invalid timeout": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "1",
			timeout:      proto.Duration{Seconds: -1},
			expErr:       true,
		},
		"missing decision policy threshold": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"decision policy with negative threshold": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "-1",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
		"decision policy with zero threshold": {
			group:        1,
			groupAccount: []byte("valid--group-address"),
			admin:        []byte("valid--admin-address"),
			comment:      "any",
			version:      1,
			threshold:    "0",
			timeout:      proto.Duration{Seconds: 1},
			expErr:       true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			m, err := NewGroupAccountInfo(
				spec.groupAccount,
				spec.group,
				spec.admin,
				spec.comment,
				spec.version,
				&ThresholdDecisionPolicy{
					Threshold: spec.threshold,
					Timeout:   spec.timeout,
				},
			)
			require.NoError(t, err)

			if spec.expErr {
				require.Error(t, m.ValidateBasic())
			} else {
				require.NoError(t, m.ValidateBasic())
			}
		})
	}
}

func TestTallyValidateBasic(t *testing.T) {
	specs := map[string]struct {
		src    Tally
		expErr bool
	}{
		"all good": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "0",
			},
		},
		"negative yes count": {
			src: Tally{
				YesCount:     "-1",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative no count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "-1",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative abstain count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "-1",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative veto count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "-1",
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTallyTotalCounts(t *testing.T) {
	specs := map[string]struct {
		src    Tally
		expErr bool
		res    string
	}{
		"all good": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			res: "4",
		},
		"negative yes count": {
			src: Tally{
				YesCount:     "-1",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative no count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "-1",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative abstain count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "-1",
				VetoCount:    "0",
			},
			expErr: true,
		},
		"negative veto count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "-1",
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			res, err := spec.src.TotalCounts()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, spec.res, math.DecimalString(res))
			}
		})
	}
}

func TestTallyAdd(t *testing.T) {
	specs := map[string]struct {
		src      Tally
		expTally Tally
		vote     Vote
		expErr   bool
		weight   string
	}{
		"add yes": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "5",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_YES},
			weight: "4",
		},
		"add no": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "2.5",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_NO},
			weight: "1.5",
		},
		"add abstain": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "2.5",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_ABSTAIN},
			weight: "1.5",
		},
		"add veto": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "2.5",
			},
			vote:   Vote{Choice: Choice_CHOICE_VETO},
			weight: "1.5",
		},
		"negative yes count": {
			src: Tally{
				YesCount:     "-1",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
			weight: "4",
		},
		"negative no count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "-1",
				AbstainCount: "0",
				VetoCount:    "0",
			},
			expErr: true,
			weight: "4",
		},
		"negative abstain count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "-1",
				VetoCount:    "0",
			},
			expErr: true,
			weight: "4",
		},
		"negative veto count": {
			src: Tally{
				YesCount:     "0",
				NoCount:      "0",
				AbstainCount: "0",
				VetoCount:    "-1",
			},
			expErr: true,
			weight: "4",
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.Add(spec.vote, spec.weight)
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, spec.expTally.YesCount, spec.src.YesCount)
				require.Equal(t, spec.expTally.NoCount, spec.src.NoCount)
				require.Equal(t, spec.expTally.AbstainCount, spec.src.AbstainCount)
				require.Equal(t, spec.expTally.VetoCount, spec.src.VetoCount)
			}
		})
	}
}

func TestTallySub(t *testing.T) {
	specs := map[string]struct {
		src      Tally
		expTally Tally
		vote     Vote
		expErr   bool
		weight   string
	}{
		"sub yes": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "0.5",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_YES},
			weight: "0.5",
		},
		"sub no": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "0.5",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_NO},
			weight: "0.5",
		},
		"sub abstain": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "0.5",
				VetoCount:    "1",
			},
			vote:   Vote{Choice: Choice_CHOICE_ABSTAIN},
			weight: "0.5",
		},
		"sub veto": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expTally: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "0.5",
			},
			vote:   Vote{Choice: Choice_CHOICE_VETO},
			weight: "0.5",
		},
		"negative yes count": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expErr: true,
			vote:   Vote{Choice: Choice_CHOICE_YES},
			weight: "2",
		},
		"negative no count": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expErr: true,
			vote:   Vote{Choice: Choice_CHOICE_NO},
			weight: "2",
		},
		"negative abstain count": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expErr: true,
			vote:   Vote{Choice: Choice_CHOICE_ABSTAIN},
			weight: "2",
		},
		"negative veto count": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expErr: true,
			vote:   Vote{Choice: Choice_CHOICE_VETO},
			weight: "2",
		},
		"unknown choice": {
			src: Tally{
				YesCount:     "1",
				NoCount:      "1",
				AbstainCount: "1",
				VetoCount:    "1",
			},
			expErr: true,
			vote:   Vote{Choice: Choice_CHOICE_UNSPECIFIED},
			weight: "2",
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.Sub(spec.vote, spec.weight)
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, spec.expTally.YesCount, spec.src.YesCount)
				require.Equal(t, spec.expTally.NoCount, spec.src.NoCount)
				require.Equal(t, spec.expTally.AbstainCount, spec.src.AbstainCount)
				require.Equal(t, spec.expTally.VetoCount, spec.src.VetoCount)
			}
		})
	}
}