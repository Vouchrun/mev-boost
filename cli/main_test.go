package cli

import (
	"context"
	"math/big"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/common"
	"github.com/flashbots/mev-boost/config"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestFloatEthTo256Wei(t *testing.T) {
	// test with small input
	i := 0.000000000000012345
	weiU256, err := common.FloatEthTo256Wei(i)
	require.NoError(t, err)
	require.Equal(t, types.IntToU256(12345), *weiU256)

	// test with zero
	i = 0
	weiU256, err = common.FloatEthTo256Wei(i)
	require.NoError(t, err)
	require.Equal(t, types.IntToU256(0), *weiU256)

	// test with large input
	i = 987654.3
	weiU256, err = common.FloatEthTo256Wei(i)
	require.NoError(t, err)

	r := big.NewInt(9876543)
	r.Mul(r, big.NewInt(1e17))
	referenceWeiU256 := new(types.U256Str)
	err = referenceWeiU256.FromBig(r)
	require.NoError(t, err)

	require.Equal(t, *referenceWeiU256, *weiU256)
}

func TestSetupGenesisPulsechainFlag(t *testing.T) {
	originalSlotTimeSec := config.SlotTimeSec
	t.Cleanup(func() { config.SlotTimeSec = originalSlotTimeSec })
	config.SlotTimeSec = common.SlotTimeSecMainnet

	var (
		genesisForkVersion string
		genesisTime        uint64
	)
	cmd := &cli.Command{
		Flags: flags,
		Action: func(_ context.Context, c *cli.Command) error {
			genesisForkVersion, genesisTime = setupGenesis(c)
			return nil
		},
	}

	err := cmd.Run(context.Background(), []string{"mev-boost", "--pulsechain"})
	require.NoError(t, err)
	require.Equal(t, genesisForkVersionPulsechain, genesisForkVersion)
	require.Equal(t, uint64(genesisTimePulsechain), genesisTime)
	require.Equal(t, uint64(common.SlotTimeSecPulsechain), config.SlotTimeSec)
}

func TestSetupGenesisMainnetDefaultSlotTime(t *testing.T) {
	originalSlotTimeSec := config.SlotTimeSec
	t.Cleanup(func() { config.SlotTimeSec = originalSlotTimeSec })
	config.SlotTimeSec = common.SlotTimeSecMainnet

	var (
		genesisForkVersion string
		genesisTime        uint64
	)
	cmd := &cli.Command{
		Flags: flags,
		Action: func(_ context.Context, c *cli.Command) error {
			genesisForkVersion, genesisTime = setupGenesis(c)
			return nil
		},
	}

	err := cmd.Run(context.Background(), []string{"mev-boost", "--mainnet"})
	require.NoError(t, err)
	require.Equal(t, genesisForkVersionMainnet, genesisForkVersion)
	require.Equal(t, uint64(genesisTimeMainnet), genesisTime)
	require.Equal(t, uint64(common.SlotTimeSecMainnet), config.SlotTimeSec)
}
