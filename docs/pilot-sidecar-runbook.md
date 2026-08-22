# Pilot Sidecar Runbook — mev-boost on 2-3 pilot trust validator hosts

**Audience:** the owner, executing by hand on the pilot validator hosts.
**Goal:** add mev-boost sidecars to **2-3 pilot trust validator hosts** (Vouch-operated) once
the relay is live at `boost-relay.vouch.run`, and measure the MEV uplift vs a control group.
**Source of truth:** this fork's `pulse` branch (the `-pulsechain` flag, commit `51c4164` +
follow-ups) and the workspace plan `knowledge/mev-stack-overview.md` §2 (sidecar) and §7
(acceptance gates). Every flag below was verified against this fork's `cli/flags.go` and
`cli/main.go`.

> **Why our fork's image and no other:** the `-pulsechain` flag exists **only** in this fork.
> **No stock mev-boost image has it** — you must build the image from this fork's `pulse`
> branch. A stock image will not set the PulseChain genesis fork version (`0x00000369`),
> genesis time, or 10s slot time and will not work on PulseChain.

---

## 1. Prerequisites

- [ ] **Relay live** at `https://boost-relay.vouch.run` with TLS (see the relay-deploy
      runbook) and passing `/eth/v1/builder/status` (HTTP 200).
- [ ] **Relay pubkey** for the `-relay` URL. Obtain it from the relay api process startup
      log: `Using BLS key: 0x...` (or from the relay website config page). The sidecar relay
      URL is `https://<RELAY_PUBKEY>@boost-relay.vouch.run`.
- [ ] **Pilot host selection:** 2-3 trust validator hosts, Vouch-operated. Per the plan
      (§8 Phase 0), cover **both** client families if possible — at least one Lighthouse-Pulse
      host and one Prysm-Pulse host.
- [ ] Confirm each host's consensus client version supports the builder API (Lighthouse-Pulse
      `--builder`, Prysm-Pulse `--http-mev-relay`/`--enable-builder`).
- [ ] Confirm the relay's protocol fee-recipient enforcement is enabled and that every pilot
      validator's registered fee recipient is VFD (it already is by protocol rule) — a
      non-VFD registration would be rejected by the relay.

---

## 2. Build the sidecar image

From the mev-boost repo checkout, on the `pulse` branch:

```bash
cd /opt/mev/mev-boost        # checkout of Vouchrun/mev-boost, branch pulse
git checkout pulse
git log -1 --oneline         # expect 51c4164 (or a later commit on pulse)
docker build -t vouchrun/mev-boost:pulse .
```

- Build from a **committed** state; tag with the commit (e.g.
  `vouchrun/mev-boost:pulse-<commit>`), never `latest` (plan §7.1).
- Push to the registry the hosts pull from, or load the image on each pilot host.

---

## 3. Per-host docker-compose

`/opt/mev/docker-compose.yml` on each pilot host:

```yaml
services:
  mev-boost:
    image: vouchrun/mev-boost:pulse-<commit>
    container_name: mev-boost
    restart: unless-stopped
    network_mode: host          # beacon node reaches it at localhost:18550
    command:
      - -pulsechain
      - -relay-check
      - -min-bid=0.06
      - -relay=https://<RELAY_PUBKEY>@boost-relay.vouch.run
    logging:
      driver: json-file
      options: { max-size: 10m, max-file: 3 }
```

Flag notes (verified against `cli/flags.go` / `cli/main.go` on the `pulse` branch):
- `-pulsechain` — sets genesis fork version `0x00000369`, genesis time `1683785555`, and 10s
  slot time (`cli/main.go` `setupGenesis`). **Required.**
- `-relay` — single relay URL `scheme://pubkey@host`; repeatable/comma-separated
  (`cli/flags.go` `relaysFlag`, alias `-relays`). Point **only** at
  `boost-relay.vouch.run`.
- `-relay-check` — checks relay status on startup and on the status API call
  (`cli/flags.go` `relayCheckFlag`).
- `-min-bid=0.06` — floor so trivial bids fall back to local block building (protects
  against relay latency games).
- `-addr` defaults to `localhost:18550` (`BOOST_LISTEN_ADDR`); `network_mode: host` means the
  beacon node reaches the sidecar at `http://localhost:18550`.
- Optional: `-metrics` / `-metrics-addr` (default `localhost:18551`) for the Prometheus
  metrics the monitoring section uses.

Start it:

```bash
docker compose up -d
docker compose logs -f mev-boost
```

---

## 4. Consensus client changes (the only host changes)

**Lighthouse-Pulse:**

```bash
# beacon node: add
lighthouse bn --builder http://localhost:18550 ...

# validator client: add
lighthouse vc --builder-proposals --gas-limit 45000000 ...
```

**Prysm-Pulse:**

```bash
# beacon node: add
--http-mev-relay=http://localhost:18550

# validator client: add
--enable-builder --suggested-gas-limit=45000000
```

> **CRITICAL — the gas-limit flag is mandatory.** Both Pulse consensus clients default to a
> **30M** gas limit while the network runs at **~45M** (elastic, ±1/1024 per block). Without
> the explicit gas-limit flag, every mev-boost block would be built at 30M — **~33% wasted
> block space per proposal** (plan §6 gas-limit-drift alert). This is the single most common
> pilot mistake; verify it on both the beacon node and the validator client.

---

## 5. Rolling procedure (one host at a time)

1. **Pick one host**; do not touch the others yet (they form the control group, §9).
2. **Check no proposal duty is imminent** for that host's validators before restarting:
   query the beacon node's proposer duties for the current/next epoch
   (`/eth/v1/validator/duties/proposer/{epoch}`) and confirm no duty within the next few
   minutes. If a duty is imminent, wait for the slot to pass.
3. **Restart order:** start the mev-boost sidecar first, then restart the **beacon node**
   (with `--builder`), then the **validator client** (with `--builder-proposals` +
   `--gas-limit`). The sidecar must be up before the beacon node points at it.
4. Confirm the sidecar's startup log shows the relay health check passing (§6.1).
5. **Watch the first post-change proposal:** confirm it is delivered via the relay (relay
   logs / data API) and that the block pays VFD with a ~45M gas limit (§6.3).
6. Wait one epoch, then proceed to the next host.

---

## 6. Verification per host

1. **Sidecar startup:** the log shows the relay check result — with `-relay-check` the
   sidecar logs whether the relay passed the health check; a failing relay is logged
   (`no relay passed the health-check!`). Expect the relay at `boost-relay.vouch.run` to
   pass.
2. **Registration visible:** after the validator client restarts, it re-registers its
   validators with the relay. Confirm the registrations appear in the relay's logs and/or
   the relay data API:
   ```
   curl -s "https://boost-relay.vouch.run/relay/v1/data/validator_registration?pubkey=<PILOT_PUBKEY>"
   ```
   (Also confirms the relay's protocol fee-recipient enforcement accepted the VFD
   registration — a non-VFD recipient would be rejected.)
3. **First delivered block:** for the host's first post-change proposal, confirm from the
   relay bid traces:
   ```
   curl -s "https://boost-relay.vouch.run/relay/v1/data/bidtraces/proposer_payload_delivered?proposer_pubkey=<PILOT_PUBKEY>"
   ```
   - `proposer_fee_recipient == 0x9325008eE3B5982c10010C8f12b6CD4943F48fA6` (VFD), and
   - `gas_limit` ≈ 45,000,000 (the ~45M network target — proves the gas-limit flag took
     effect; a 30M gas limit means the flag was missed).

---

## 7. Failure drill

1. `docker compose stop mev-boost` on a pilot host (mid-slot is fine — this is the drill).
2. Confirm the beacon node/validator client **falls back to local block building with zero
   missed proposals** (no relay bid → local block; priority fees still land at VFD). Check
   the beacon/VC logs for the slot.
3. Restart the sidecar: `docker compose start mev-boost`; confirm it reconnects to the relay
   and the next proposal is delivered again.
4. Record the outcome. A missed proposal during the drill fails the §8 missed-proposal gate.

> Local fallback is inherent to the design: when no relay bid is returned, the validator
> builds locally as today. No keys or fee-recipient changes are involved.

---

## 8. Monitoring during the pilot (plan §7 gates)

- [ ] **Missed-proposal watcher:** compare each pilot host's proposer duties vs blocks
  actually delivered, per validator, and alert if a pilot host drops below the fleet
  baseline (plan §6). Gate: pilot missed-proposal rate ≤ non-pilot fleet baseline (§7.4).
- [ ] **Gas-limit drift alert:** alert if any delivered block's gas limit diverges from the
  ~45M elastic target (catches a missed `--gas-limit 45000000` flag — §4).
- [ ] **Fee-recipient enforcement:** 100% of relay-delivered blocks must pay the registered
  recipient (VFD for Vouch validators) — verified from bid traces + execution traces (§7.4).
- [ ] **Revenue reconciliation:** bid traces ↔ VFD balance deltas ↔ DaoDistributor inflows
  must agree (no double counting, plan deep-dive §9).

---

## 9. Pilot measurement methodology

- **Control group:** the **non-pilot trust validators** (same window) — the robust
  comparison is pilot vs control on the same days, not vs a historical point sample.
- **Per-proposal income identification:** a block whose `feeRecipient == VFD`
  (`0x9325008eE3B5982c10010C8f12b6CD4943F48fA6`) is a Vouch proposal (Vouch validators must
  register VFD by protocol rule). **Priority fees = sum over receipts of
  `(effectiveGasPrice − baseFeePerGas) × gasUsed`**.
- **Method + script location:** this is exactly what the workspace's
  `skills/beacon-rpc/scripts/measure_proposal_income.py` implements (batched/parallel block
  measurement; identifies Vouch blocks by `feeRecipient == VFD` and sums priority fees from
  receipts). Re-run it over the pilot window for the pilot group and the control group.
- **Baseline context (measured):** current per-proposal income median ≈ **11.4k PLS** (avg
  ≈ 18.4k PLS); Vouch fleet share ≈ **10.1%** of network blocks.
- **Success criterion:** pilot-group uplift vs the control group. The Ethereum analog
  suggests **+42% to +60%** staking-reward uplift from MEV-Boost; the pilot measures the
  actual PulseChain number (the first real PulseChain bid data — the Atlas relay was down).

---

## 10. Rollback

Per host:

```bash
docker compose down                       # stop + remove the sidecar container
# remove the two consensus-client flags:
#   Lighthouse:  --builder (bn) + --builder-proposals --gas-limit 45000000 (vc)
#   Prysm:       --http-mev-relay (bn) + --enable-builder --suggested-gas-limit=45000000 (vc)
# restart the beacon node + validator client
```

The host returns to **pure local block building** (priority fees → VFD), exactly as before
the pilot. No key or fee-recipient changes are ever involved.

---

## 11. Go/no-go gate for fleet rollout (Phase 1) — plan §7.4 summary

Advance to Phase 1 (fleet rollout) only if **all** of these pass with executable evidence:

- [ ] Pilot missed-proposal rate ≤ non-pilot fleet baseline (no degradation from the
      sidecar/relay path).
- [ ] 100% of relay-delivered blocks pay the registered fee recipient (VFD) — from bid
      traces + execution traces.
- [ ] **Measured uplift:** pilot-group per-proposal income materially exceeds the control
      group (the plan's go/no-go: searcher + builder proceed only if pilot bids materially
      exceed the local-build baseline; relay + sidecars continue regardless — enforcement +
      data).
- [ ] Revenue reconciliation: bid traces ↔ VFD balance deltas ↔ DaoDistributor inflows agree.

---

## TODO-OPERATOR items

- [ ] Confirm the exact relay pubkey to embed in the `-relay` URL (from the relay api startup
      log `Using BLS key: 0x...` or the relay website).
- [ ] Confirm the image tag / registry distribution path for the pilot hosts.
- [ ] Confirm the exact Lighthouse-Pulse / Prysm-Pulse client versions on the chosen hosts
      support the builder flags (plan §7.3 client matrix).
- [ ] Set up the missed-proposal watcher and gas-limit drift alert before the first pilot
      host goes live (§8).
- [ ] Record the control-group baseline window before enabling MEV-Boost on the first pilot
      host.
