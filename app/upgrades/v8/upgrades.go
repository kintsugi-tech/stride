package v8

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	claimkeeper "github.com/Stride-Labs/stride/v7/x/claim/keeper"
	claimtypes "github.com/Stride-Labs/stride/v7/x/claim/types"
)

var (
	UpgradeName             = "v8"
	EvmosAirdropDistributor = "TODO"
	EvmosAirdropIdentifier  = "evmos"
	AirdropDuration         = time.Hour * 24 * 30 * 12 * 3 // 3 years
	ResetAirdropIdentifiers = []string{"stride", "gaia", "osmosis", "juno", "stars"}
)

// CreateUpgradeHandler creates an SDK upgrade handler for v8
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	cdc codec.Codec,
	claimKeeper claimkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Starting upgrade v8...")

		// Reset Claims
		ctx.Logger().Info("Resetting airdrop claims...")
		for _, claimType := range ResetAirdropIdentifiers {
			if err := claimKeeper.ResetClaimStatus(ctx, claimType); err != nil {
				return vm, errorsmod.Wrapf(err, fmt.Sprintf("unable to reset %s claim status", claimType))
			}
		}

		// Add the evmos airdrop
		ctx.Logger().Info("Adding evmos airdrop...")
		blockTime := uint64(ctx.BlockTime().Unix())
		duration := uint64(AirdropDuration.Seconds())
		if err := claimKeeper.CreateAirdropAndEpoch(ctx, EvmosAirdropDistributor, claimtypes.DefaultClaimDenom, blockTime, duration, EvmosAirdropIdentifier); err != nil {
			return vm, err
		}

		ctx.Logger().Info("Loading airdrop allocations...")
		claimKeeper.LoadAllocationData(ctx, allocations)

		ctx.Logger().Info("Running module mogrations...")
		return mm.RunMigrations(ctx, configurator, vm)
	}
}