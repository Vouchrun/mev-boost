# PulseChain config values for the `-pulsechain` flag

Task: add a `-pulsechain` network flag to the Vouchrun/mev-boost fork. Atlas's fork
(`atlasbuilderxyz/mev-boost`, MIT) is used as a reference for the PulseChain values only;
the implementation follows upstream flashbots/mev-boost's own network-flag pattern
(`-mainnet` / `-sepolia` / `-holesky` / `-hoodi`). No commit lineage was taken from Atlas.

## Config values

| Value | Atlas value | Live beacon value | Value used | Notes |
|---|---|---|---|---|
| genesis_fork_version | `0x00000369` | `0x00000369` | `0x00000369` | matches live `/eth/v1/beacon/genesis` and `/eth/v1/config/spec` |
| genesis_time | `1683785555` | `1683785555` | `1683785555` | matches live `/eth/v1/beacon/genesis` |
| genesis_validators_root | not set in mev-boost | `0x3357ba0018a2582aeabe4ae847aa17d50a3a99aaeb66293c01f80a83aecd0c90` | not used by mev-boost | mev-boost does not consume this value; it matters for the relay's signature-domain validation (recorded here for the relay fork) |
| seconds_per_slot | `10` | `10` | `10` | `SlotTimeSecPulsechain`; matches live `/eth/v1/config/spec` |
| altair_fork_version / epoch | not set in mev-boost | `0x0000036a` / `1` | not used by mev-boost | relay-side concern |
| bellatrix_fork_version / epoch | not set in mev-boost | `0x0000036b` / `2` | not used by mev-boost | relay-side concern |
| capella_fork_version / epoch | not set in mev-boost | `0x0000036c` / `3` | not used by mev-boost | relay-side concern |
| deneb_fork_version / epoch | not set in mev-boost | `0xffffffff` / `18446744073709551615` (disabled) | not used by mev-boost | Deneb disabled on PulseChain; mev-boost handles this generically |
| block gas limit | not set in mev-boost | ~45,000,000 (elastic, ±1/1024) | not used by mev-boost | mev-boost never sets a gas limit — the validator client does (`--gas-limit 45000000` on Lighthouse-Pulse, `--suggested-gas-limit=45000000` on Prysm-Pulse) |
| chain id | — | `369` (DEPOSIT_CHAIN_ID) | not used by mev-boost | informational |

Live beacon source: `https://rpc-beacon.vouch.run/eth/v1/beacon/genesis` and
`https://rpc-beacon.vouch.run/eth/v1/config/spec` (queried 2026-08-21). Atlas values
extracted from commit `e0ba61c` ("Add PulseChain network support") on `atlas/main`.

## Files changed (branch `pulsechain`)

| File | Change |
|---|---|
| `common/common.go` | added `SlotTimeSecPulsechain = 10` constant |
| `cli/flags.go` | added `pulsechainFlag` (`-pulsechain`, env `PULSECHAIN`, GENESIS category) |
| `cli/main.go` | added `genesisForkVersionPulsechain` / `genesisTimePulsechain` constants; `setupGenesis` case sets fork version, genesis time and `config.SlotTimeSec`; updated fatal error message |
| `cli/main_test.go` | added `TestSetupGenesisPulsechainFlag` + `TestSetupGenesisMainnetDefaultSlotTime` (mainnet regression) |
| `README.md` | PulseChain mainnet usage section, TOC entry, CLI args list |
| `CONTRIBUTING.md` | flag list now includes `-pulsechain` |
| `pulsechain-values.md` | this report |

## Verified build & test results

- **Go toolchain:** `go1.24.13 windows/amd64` (portable ZIP, session-only PATH,
  no system-wide install).
- **Build:** `go build ./...` — **PASSED** (exit 0).
- **Tests:** `go test ./...` — **ALL PASSED** (exit 0). 175 tests run
  (47 top-level + 128 subtests), **0 failed, 0 skipped**.
  - `go test ./cli/...` — ok (2.918s), includes the new
    `TestSetupGenesisPulsechainFlag` and `TestSetupGenesisMainnetDefaultSlotTime`.
  - `server`, `server/mock`, `server/types` — ok.
- **No compile errors** from the `-pulsechain` change; no fix commit required.

## Uncertainties / notes

- Atlas's commit is based on upstream v1.9; our `develop` is newer, but the flag wiring is
  identical, so the port is direct (no behavioral drift).
- `genesis_validators_root` and the fork-epoch values are **not** consumed by mev-boost
  (only the relay uses them for signature-domain validation). They are recorded here as
  reference for the mev-boost-relay fork task.
- `setupGenesis` checks `-pulsechain` **before** `-mainnet` because `-mainnet` defaults to
  `true`; this ordering is required and matches Atlas.
