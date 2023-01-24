package types

import (
	"fmt"

	sdk_types "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultMaxEvidenceAgeNumBlocks = 100
	DefaultMaxBuildersPerAuction   = 5
)

var (
	DefaultAllowedBuilderAddresses = []string{}
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		MaxEvidenceAgeNumBlocks: DefaultMaxEvidenceAgeNumBlocks,
		MaxBuildersPerAuction:   DefaultMaxBuildersPerAuction,
		AllowedBuilderAddresses: DefaultAllowedBuilderAddresses,
	}
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.MaxEvidenceAgeNumBlocks < 1 {
		return fmt.Errorf("max_evidence_age_num_blocks=%d < 1", p.MaxEvidenceAgeNumBlocks)
	}

	if p.MaxBuildersPerAuction < 1 {
		return fmt.Errorf("max_builders_per_auction=%d < 1", p.MaxBuildersPerAuction)
	}

	for _, addr := range p.AllowedBuilderAddresses {
		_, err := sdk_types.AccAddressFromBech32(addr)
		if err != nil {
			return fmt.Errorf("allowed_builder_addresses: %w", err)
		}
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
