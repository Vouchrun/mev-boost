# mev-boost on PulseChain: operator setup guide

Reference deployment for connecting a PulseChain validator to the Vouch MEV
relay (`boost-relay.vouch.run`). The sidecar polls the relay for block headers
per slot and submits signed blinded blocks when your validator is selected to
propose, so your block building is outsourced to MEV builders competing via
the relay. This is the setup guide linked from the top of the repository
README.

## Prerequisites

- A synced PulseChain beacon node + validator client (Lighthouse-Pulse or
  Prysm-Pulse).
- The beacon node must point its builder endpoint at the sidecar (default port
  **18550**):
  - Lighthouse-Pulse: `--builder http://localhost:18550` on the beacon node and
    `--builder-proposals --gas-limit 45000000` on the validator client.
  - Prysm-Pulse: `--http-mev-relay=http://localhost:18550` (beacon node) and
    `--enable-builder --suggested-gas-limit=45000000` (validator client).
- Outbound HTTPS only (to the relay); no inbound ports needed.

> **CRITICAL - set the gas limit explicitly.** Both Pulse consensus clients
> default to a **30M** gas limit while the network runs at **~45M** (elastic,
> +/-1/1024 per block). Without the explicit gas-limit flag, every mev-boost
> block is built at 30M - that is **~33% of block space wasted on every
> proposal**. Verify the flag on BOTH the beacon node and the validator client.

## Quickstart

```bash
cd ops
docker compose up -d
```

Verify:

- `docker compose logs -f mev-boost` shows the relay check as **OK** at startup.
- The validator client automatically sends its registration to the relay each
  epoch - look for registration-related log lines from your validator client.
- Run the relay-side checks in [Verify it is working](#verify-it-is-working).

## Apply and restart order

1. **Check no proposer duty is imminent** before restarting anything: query the
   beacon node's proposer duties for the current/next epoch
   (`/eth/v1/validator/duties/proposer/{epoch}`) and wait for any duty slot to
   pass.
2. Start the mev-boost sidecar first: `docker compose up -d`.
3. Restart the **beacon node** (with the builder endpoint flag).
4. Restart the **validator client** (with the builder + gas-limit flags).

The sidecar must be up before the beacon node points at it.

## Verify it is working

1. **Sidecar startup:** with `-relay-check` (in the compose file) the log shows
   whether the relay passed its health check; a failing relay is logged
   (`no relay passed the health-check!`). The relay at `boost-relay.vouch.run`
   should pass.
2. **Registration visible:** the validator client re-registers its validators
   with the relay every epoch. Confirm the registration via the relay data API
   (using your validator's pubkey):
   ```bash
   curl -s "https://boost-relay.vouch.run/relay/v1/data/validator_registration?pubkey=<YOUR_VALIDATOR_PUBKEY>"
   ```
3. **First delivered block:** after your first post-change proposal, check the
   relay bid traces:
   ```bash
   curl -s "https://boost-relay.vouch.run/relay/v1/data/bidtraces/proposer_payload_delivered?proposer_pubkey=<YOUR_VALIDATOR_PUBKEY>"
   ```
   - `gas_limit` is ~45,000,000 - this proves the gas-limit flag took effect
     (a 30M gas limit means it was missed);
   - `proposer_fee_recipient` is the fee recipient configured in your validator
     client.
4. **No curl?** Browse `https://boost-relay.vouch.run/mevblocks` - the public
   delivered-blocks explorer.

## Notes

- Registrations carry the **fee recipient configured in the validator client**;
  MEV payouts from winning bids go to that address. Keep it set to your own
  address (or your protocol's collector if applicable).
- The compose file ships `-min-bid=2000`, a 2000 PLS floor (the flag is
  denominated in PLS units, converted to wei = 2000e18). Slots whose best relay
  bid is below the floor build locally instead - your validator keeps proposing
  either way. Set `-min-bid=0` to accept any bid.
- The getHeader timeout is set to **3000ms** because the relay is reached over
  WAN with a cold TLS handshake per call; if your validators are LAN-close to a
  relay you can lower it via `-request-timeout-getheader` (ms).
- If the relay is unreachable, the sidecar fails closed: your beacon node
  continues producing blocks locally with zero missed proposals.

## WAN relays: raising the getHeader budget

There are **two** getHeader timeouts:

- `-request-timeout-getheader` - the HTTP client timeout, already set to 3000ms
  in the compose file above.
- `timeout_get_header_ms` - the per-request **context budget** that mev-boost
  actually enforces (`min(timeout_get_header_ms, late_in_slot_time_ms -
  ms_into_slot)`). It defaults to **950ms** and is configurable **only** via a
  YAML config file - the CLI flag above does not touch it.

When the relay is a WAN hop away, the 950ms budget kills every getHeader (cold
TLS alone is ~1.2s), and your validator silently falls back to building blocks
locally. If you see that happening, switch to the config file:

- Use `mev-boost-config.example.yaml` in this directory as the starting point.
- Compose wiring: remove the `-relay=...` command line entry (`-relay` and the
  config `relays:` list are mutually exclusive), add
  `- -config=/etc/mev-boost/config.yaml` and a volume mount
  `./config.yaml:/etc/mev-boost/config.yaml:ro`.

Validators on a LAN-close path do not need this - the 950ms default is fine
there, and the compose above (with the CLI flag only) is all you need.

**Relay keep-alive.** Beacon nodes (the lighthouse builder client) apply a
hardcoded **1s** cutoff to getHeader; on a WAN path a cold TLS handshake
(~1.2s) exceeds it, so every getHeader times out even after the budget above is
raised. Periodic status pings keep the TLS transport warm, so getHeader rides
an established connection and answers in ~0.2-0.4s. This is on by default:
`-relay-keepalive-ms` (default **30000ms**, config-file equivalent
`relay_keepalive_ms`; `-relay-keepalive-ms=0` disables it). It requires an
image built from the `pulse` branch at or after commit `7ba09a3` - the current
`ghcr.io/vouchrun/mev-boost:pulse` image includes it.

## Optional: Prometheus metrics

`-metrics` is already in the compose file; metrics are served on
`localhost:18551` (loopback only). To scrape them, add a job to your Prometheus
configuration:

```yaml
   - job_name: 'mev_boost'
     metrics_path: /metrics
     static_configs:
       - targets: ['localhost:18551']
```

## Rollback

```bash
docker compose down
```

Then remove the builder and gas-limit flags from the beacon node and validator
client and restart both. The validator returns to pure local block building; no
key or fee-recipient changes are involved.
