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
- `-min-bid=2000` (2000 wei) only accepts bids above a trivial floor; if the
  relay returns no qualifying bid the beacon node falls back to building the
  block locally — your validator keeps proposing either way.
- If the relay is unreachable, the sidecar fails closed: your beacon node
  continues producing blocks locally with zero missed proposals.
