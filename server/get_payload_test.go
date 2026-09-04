// get_payload_test.go
package server

import (
	"os"
	"testing"

	eth2Api "github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/stretchr/testify/require"
)

// TestDecodeSignedBlindedBeaconBlock_MissingEthConsensusVersionDefaultCapella
// verifies that a Capella signed blinded beacon block decodes successfully even
// when the Eth-Consensus-Version header is absent. Production lighthouse-pulse
// v2.4.2 omits the header on blinded block submissions; PulseChain is
// Capella-era with Deneb permanently disabled, so Capella is the only possible
// version. Pre-fix this request failed with ErrMissingEthConsensusVersion and
// the proposer missed the slot.
func TestDecodeSignedBlindedBeaconBlock_MissingEthConsensusVersionDefaultCapella(t *testing.T) {
	jsonBytes, err := os.ReadFile("../testdata/signed-blinded-beacon-block-capella.json")
	require.NoError(t, err)

	t.Run("JSON", func(t *testing.T) {
		block := new(eth2Api.VersionedSignedBlindedBeaconBlock)
		err := decodeSignedBlindedBeaconBlock(jsonBytes, MediaTypeJSON, "", block)
		require.NoError(t, err)
		require.Equal(t, spec.DataVersionCapella, block.Version)
		require.NotNil(t, block.Capella)
		blockHash, err := block.ExecutionBlockHash()
		require.NoError(t, err)
		require.NotEmpty(t, blockHash)
	})

	t.Run("SSZ", func(t *testing.T) {
		block := new(eth2Api.VersionedSignedBlindedBeaconBlock)
		err := decodeSignedBlindedBeaconBlock(jsonBytes, MediaTypeJSON, "", block)
		require.NoError(t, err)
		sszBytes, err := block.Capella.MarshalSSZ()
		require.NoError(t, err)

		decoded := new(eth2Api.VersionedSignedBlindedBeaconBlock)
		err = decodeSignedBlindedBeaconBlock(sszBytes, MediaTypeOctetStream, "", decoded)
		require.NoError(t, err)
		require.Equal(t, spec.DataVersionCapella, decoded.Version)
		require.NotNil(t, decoded.Capella)
	})
}
