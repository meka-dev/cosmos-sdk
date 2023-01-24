package types

import (
	"fmt"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Builders:  []Builder{},
		Proposers: []Proposer{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Check for duplicated address in builder
	builderAddressMap := make(map[string]struct{})

	for _, elem := range gs.Builders {
		address := string(BuilderStoreKey(elem.Address))
		if _, ok := builderAddressMap[address]; ok {
			return fmt.Errorf("duplicated address for builder")
		}
		builderAddressMap[address] = struct{}{}
	}
	// Check for duplicated address in proposer
	proposerAddressMap := make(map[string]struct{})

	for _, elem := range gs.Proposers {
		address := string(ProposerStoreKey(elem.Address))
		if _, ok := proposerAddressMap[address]; ok {
			return fmt.Errorf("duplicated address for proposer")
		}
		proposerAddressMap[address] = struct{}{}
	}

	return nil
}
