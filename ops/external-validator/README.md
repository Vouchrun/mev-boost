# External validator: mev-boost sidecar for the Vouch relay

Reference deployment for connecting a PulseChain validator to the Vouch MEV
relay (`boost-relay.vouch.run`). The sidecar polls the relay for block headers
per slot and submits signed blinded blocks when your validator is selected to
propose, so your block building is outsourced to MEV builders competing via
the relay.

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

## Quickstart

```bash
cd ops/external-validator
docker compose up -d
```

Verify:

- `docker compose logs -f mev-boost` shows the relay check as **OK** at startup.
- The validator client automatically sends its registration to the relay each
  epoch — look for registration-related log lines from your validator client.

## Notes

- Registrations carry the **fee recipient configured in the validator client**;
  MEV payouts from winning bids go to that address. Keep it set to your own
  address (or your protocol's collector if applicable).
- `-min-bid=2000` sets a 2000 PLS floor (the flag is denominated in PLS units,
  converted to wei = 2000e18). Slots whose best relay bid is below the floor
  build locally instead — your validator keeps proposing either way. Set
  `-min-bid=0` to accept any bid.
- The getHeader timeout is set to **3000ms** because the relay is reached over
  WAN with a cold TLS handshake per call; if your validators are LAN-close to a
  relay you can lower it via `-request-timeout-getheader` (ms).
- If the relay is unreachable, the sidecar fails closed: your beacon node
  continues producing blocks locally with zero missed proposals.

## WAN relays: raising the getHeader budget

There are **two** getHeader timeouts:

- `-request-timeout-getheader` — the HTTP client timeout, already set to 3000ms
  in the compose file above.
- `timeout_get_header_ms` — the per-request **context budget** that mev-boost
  actually enforces (`min(timeout_get_header_ms, late_in_slot_time_ms -
  ms_into_slot)`). It defaults to **950ms** and is configurable **only** via a
  YAML config file — the CLI flag above does not touch it.

When the relay is a WAN hop away, the 950ms budget kills every getHeader (cold
TLS alone is ~1.2s), and your validator silently falls back to building blocks
locally. If you see that happening, switch to the config file:

- Use `mev-boost-config.example.yaml` in this directory as the starting point.
- Compose wiring: remove the `-relay=...` command line entry (`-relay` and the
  config `relays:` list are mutually exclusive), add
  `- -config=/etc/mev-boost/config.yaml` and a volume mount
  `./config.yaml:/etc/mev-boost/config.yaml:ro`.

Validators on a LAN-close path do not need this — the 950ms default is fine
there, and the compose above (with the CLI flag only) is all you need.
