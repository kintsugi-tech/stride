package v14_test

import (
	"fmt"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	evmosvestingtypes "github.com/evmos/vesting/x/vesting/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	claimtypes "github.com/Stride-Labs/stride/v13/x/claim/types"

	"github.com/Stride-Labs/stride/v13/app/apptesting"
	v14 "github.com/Stride-Labs/stride/v13/app/upgrades/v14"
)

var (
	emptyCoins         = sdk.Coins{}
	dummyUpgradeHeight = int64(5)
	// Shortly after the upgrade - 9/25/23
	AfterUpgrade = int64(1695677732)
	Account2End  = int64(1820016452)
	InitCoins    = int64(100)

	osmoAirdropId = "osmosis"
	ustrd         = "ustrd"
)

type UpgradeTestSuite struct {
	apptesting.AppTestHelper
}

func (s *UpgradeTestSuite) SetupTest() {
	s.Setup()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (s *UpgradeTestSuite) TestUpgrade() {
	// Setup
	s.SetupAirdrops()
	s.SetupVestingStoreBeforeUpgrade()
	s.FundConsToSendToProviderModuleAccount()

	// Upgrade
	s.ConfirmUpgradeSucceededs("v14", dummyUpgradeHeight)

	// Post-upgrade checks
	s.CheckVestingStoreAfterUpgrade()
	s.CheckCcvConsumerParamsAfterUpgrade()
	s.CheckRefundAfterUpgrade()
	s.CheckAirdropsInitialized()
}

func (s *UpgradeTestSuite) FundConsToSendToProviderModuleAccount() {
	// Fund the cons_to_send_to_provider module account
	address := sdk.MustAccAddressFromBech32(v14.ConsToSendToProvider)
	s.FundAccount(address, sdk.NewCoin(s.App.StakingKeeper.BondDenom(s.Ctx), sdkmath.NewInt(InitCoins)))
}

func (s *UpgradeTestSuite) CheckRefundAfterUpgrade() {
	afterCtx := s.Ctx.WithBlockHeight(dummyUpgradeHeight)
	// Verify the correct number of tokens were sent out of the cons_to_send_to_provider module account
	icsFeeAddress := sdk.MustAccAddressFromBech32(v14.ConsToSendToProvider)
	// Check the account balance
	balance := s.App.BankKeeper.GetBalance(afterCtx, icsFeeAddress, s.App.StakingKeeper.BondDenom(afterCtx))
	refundFrac, err := sdk.NewDecFromStr(v14.RefundFraction)
	s.Require().NoError(err)
	remainingFrac := sdk.NewDec(int64(1)).Sub(refundFrac)
	expectedNumCoins := remainingFrac.Mul(sdk.NewDec(InitCoins)).TruncateInt64()
	s.Require().Equal(sdk.NewInt64Coin(s.App.StakingKeeper.BondDenom(s.Ctx), expectedNumCoins), balance)

}

func (s *UpgradeTestSuite) CheckCcvConsumerParamsAfterUpgrade() {
	afterCtx := s.Ctx.WithBlockHeight(dummyUpgradeHeight)
	// Verify the ccv consumer params are set correctly
	ccvConsumerParams := s.App.ConsumerKeeper.GetConsumerParams(afterCtx)
	// Verify DistributionTransmissionChannel is set
	s.Require().Equal(v14.DistributionTransmissionChannel, ccvConsumerParams.DistributionTransmissionChannel)
	// Verify ProviderFeePoolAddrStr is set
	s.Require().Equal(v14.ProviderFeePoolAddrStr, ccvConsumerParams.ProviderFeePoolAddrStr)
	// Verify ConsumerRedistributionFraction is set
	s.Require().Equal(v14.ConsumerRedistributionFraction, ccvConsumerParams.ConsumerRedistributionFraction)
	// Verify Enabled is set
	s.Require().Equal(v14.Enabled, ccvConsumerParams.Enabled)

	// TODO: verify reward denoms are set correctly
}

func (s *UpgradeTestSuite) SetupVestingStoreBeforeUpgrade() {
	// Initialize the two accounts as continuous vesting accounts
	// Create the ContinuousVestingAccount
	address1, err := sdk.AccAddressFromBech32(v14.Account1)
	s.Require().NoError(err)
	address2, err := sdk.AccAddressFromBech32(v14.Account2)
	s.Require().NoError(err)
	account1 := s.CreateContinuousVestingAccount(address1, v14.VestingStartTimeAccount1, v14.VestingEndTimeAccount1, v14.Account1VestingUstrd)
	account2 := s.CreateContinuousVestingAccount(address2, v14.VestingStartTimeAccount2, v14.VestingEndTimeAccount2, v14.Account2VestingUstrd)

	// Fund accounts 1 and 2
	s.FundAccount(address1, sdk.NewCoin(s.App.StakingKeeper.BondDenom(s.Ctx), sdkmath.NewInt(v14.Account1VestingUstrd)))
	s.FundAccount(address2, sdk.NewCoin(s.App.StakingKeeper.BondDenom(s.Ctx), sdkmath.NewInt(v14.Account2VestingUstrd)))

	// Store the accounts as ContinuousVestingAccounts
	s.App.AccountKeeper.SetAccount(s.Ctx, account1)
	s.App.AccountKeeper.SetAccount(s.Ctx, account2)
}

func (s *UpgradeTestSuite) CheckVestingStoreAfterUpgrade() {
	afterCtx := s.Ctx.WithBlockHeight(dummyUpgradeHeight)
	address1, err := sdk.AccAddressFromBech32(v14.Account1)
	s.Require().NoError(err)
	address2, err := sdk.AccAddressFromBech32(v14.Account2)
	s.Require().NoError(err)
	// Verify account1 is now a ClawbackVestingAccount
	account1 := s.App.AccountKeeper.GetAccount(afterCtx, address1)
	s.Require().IsType(&evmosvestingtypes.ClawbackVestingAccount{}, account1)

	// And that no tokens are vested after the upgrade
	vestingAccount1 := account1.(*evmosvestingtypes.ClawbackVestingAccount)
	afterUpgrade := time.Unix(AfterUpgrade, 0)
	coins := vestingAccount1.GetVestedOnly(afterUpgrade)
	s.Require().Equal(int64(0), coins.AmountOf(s.App.StakingKeeper.BondDenom(s.Ctx)).Int64())

	// Verify account2 is still a ContinuousVestingAccount
	account2 := s.App.AccountKeeper.GetAccount(afterCtx, address2)
	s.Require().IsType(&types.ContinuousVestingAccount{}, account2)
	// Verify the correct number of tokens is vested after the upgrade
	vestingAccount2 := account2.(*types.ContinuousVestingAccount)
	coins = vestingAccount2.GetVestedCoins(afterUpgrade)
	expectedVestedCoins := int64(float64(v14.Account2VestingUstrd)*(float64(AfterUpgrade-v14.VestingStartTimeAccount2)/float64(v14.VestingEndTimeAccount2-v14.VestingStartTimeAccount2))) + 1 // add 1, rounding
	s.Require().Equal(expectedVestedCoins, coins.AmountOf(s.App.StakingKeeper.BondDenom(s.Ctx)).Int64())
}

// ---------------------- Utils ----------------------
func initBaseAccount(address sdk.AccAddress) *authtypes.BaseAccount {
	bacc := authtypes.NewBaseAccountWithAddress(address)
	return bacc
}

func (s *UpgradeTestSuite) CreateContinuousVestingAccount(address sdk.AccAddress, start int64, end int64, coins int64) *types.ContinuousVestingAccount {
	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)

	// init a base account
	// send tokens to the base account
	bacc := initBaseAccount(address)
	origCoins := sdk.Coins{sdk.NewInt64Coin(s.App.StakingKeeper.BondDenom(s.Ctx), coins)}
	cva := types.NewContinuousVestingAccount(bacc, origCoins, start, end)

	// Sanity check the vesting schedule
	// require no coins vested in the very beginning of the vesting schedule
	vestedCoins := cva.GetVestedCoins(startTime)
	s.Require().Nil(vestedCoins)

	// require all coins vested at the end of the vesting schedule)
	vestedCoins = cva.GetVestedCoins(endTime)
	s.Require().Equal(origCoins, vestedCoins)

	// require 50% of coins vested
	midpoint := time.Unix((start+end)/2, 0)
	vestedCoins = cva.GetVestedCoins(midpoint)
	s.Require().Equal(sdk.Coins{sdk.NewInt64Coin(s.App.StakingKeeper.BondDenom(s.Ctx), coins/2)}, vestedCoins)

	return cva
}

func (s *UpgradeTestSuite) SetupAirdrops() {
	// Add a test aidrop to the store
	params := claimtypes.Params{
		Airdrops: []*claimtypes.Airdrop{
			{
				AirdropIdentifier: osmoAirdropId,
				ClaimedSoFar:      sdkmath.NewInt(1000000),
			},
		},
	}
	err := s.App.ClaimKeeper.SetParams(s.Ctx, params)
	s.Require().NoError(err, "no error expected when setting claim params")
	// Set vesting to 0s
	claimtypes.DefaultVestingInitialPeriod, err = time.ParseDuration("0s")
	s.Require().NoError(err, "no error expected when setting vesting initial period")
}

func (s *UpgradeTestSuite) CheckAirdropsInitialized() {
	afterCtx := s.Ctx.WithBlockHeight(dummyUpgradeHeight)

	// Check that all airdrops were added, osmosis airdrop wasn't removed
	claimParams, err := s.App.ClaimKeeper.GetParams(s.Ctx)
	s.Require().NoError(err, "no error expected when getting params")
	s.Require().Len(claimParams.Airdrops, 5, "there should be exactly 5 airdrops")

	// ------ OSMO -------
	osmoAirdrop := claimParams.Airdrops[0]
	s.Require().Equal(osmoAirdropId, osmoAirdrop.AirdropIdentifier, "osmo airdrop identifier") // verify this wasn't deleted

	// ------ INJECTIVE -------
	injectiveAirdrop := claimParams.Airdrops[1]
	s.CheckAirdropAdded(afterCtx, injectiveAirdrop, v14.InjectiveAirdropDistributor, v14.InjectiveAirdropIdentifier, v14.InjectiveChainId, true)

	// ------ COMDEX -------
	comdexAirdrop := claimParams.Airdrops[2]
	s.CheckAirdropAdded(afterCtx, comdexAirdrop, v14.ComdexAirdropDistributor, v14.ComdexAirdropIdentifier, v14.ComdexChainId, false)

	// ------ SOMM -------
	sommAirdrop := claimParams.Airdrops[3]
	s.CheckAirdropAdded(afterCtx, sommAirdrop, v14.SommAirdropDistributor, v14.SommAirdropIdentifier, v14.SommChainId, false)

	// ------ UMEE -------
	umeeAirdrop := claimParams.Airdrops[4]
	s.CheckAirdropAdded(afterCtx, umeeAirdrop, v14.UmeeAirdropDistributor, v14.UmeeAirdropIdentifier, v14.UmeeChainId, false)
}

func (s *UpgradeTestSuite) CheckAirdropAdded(ctx sdk.Context, airdrop *claimtypes.Airdrop, distributor string, identifier string, chainId string, autopilotEnabled bool) {
	// Check that the params of the airdrop were initialized
	s.Require().Equal(identifier, airdrop.AirdropIdentifier, fmt.Sprintf("%s airdrop identifier", identifier))
	s.Require().Equal(chainId, airdrop.ChainId, fmt.Sprintf("%s airdrop chain-id", identifier))
	s.Require().Zero(airdrop.ClaimedSoFar.Int64(), fmt.Sprintf("%s claimed so far", identifier))
	s.Require().Equal(distributor, airdrop.DistributorAddress, fmt.Sprintf("%s airdrop distributor", identifier))
	s.Require().Equal(v14.AirdropDuration, airdrop.AirdropDuration, fmt.Sprintf("%s airdrop duration", identifier))
	s.Require().Equal(ustrd, airdrop.ClaimDenom, fmt.Sprintf("%s airdrop claim denom", identifier))
	s.Require().Equal(v14.AirdropStartTime, airdrop.AirdropStartTime, fmt.Sprintf("%s airdrop start time", identifier))
	s.Require().Equal(autopilotEnabled, airdrop.AutopilotEnabled, fmt.Sprintf("%s airdrop autopilot enabled", identifier))

	claimRecords := s.App.ClaimKeeper.GetClaimRecords(ctx, identifier)
	s.Require().Positive(len(claimRecords), fmt.Sprintf("there should be at least one claim record for %s", identifier))

	// Check that an epoch was created
	epochInfo, found := s.App.EpochsKeeper.GetEpochInfo(ctx, fmt.Sprintf("airdrop-%s", identifier))
	s.Require().True(found, "epoch tracker should be found")
	s.Require().Zero(epochInfo.CurrentEpoch, "epoch should be zero")
	s.Require().Equal(epochInfo.Duration, claimtypes.DefaultEpochDuration, "epoch duration should be equal to airdrop duration")
}