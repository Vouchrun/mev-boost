# Security Policy

We appreciate any contributions and responsible disclosures, and will make
every effort to acknowledge your contributions.

## Supported Versions

This fork does not publish versioned releases. Use the container image built
from the `pulse` branch (`ghcr.io/vouchrun/mev-boost:pulse`) or build from
source.

## Reporting a Vulnerability

To report a vulnerability, open an issue at
https://github.com/Vouchrun/mev-boost/issues and provide all the necessary
details to reproduce it, such as:

- Commit / image reference
- Operating System
- Consensus / Execution client combination and version
- Network (PulseChain mainnet or other)

Please include the steps to reproduce it using as much detail as possible with
the corresponding logs from `mev-boost` and/or logs from the consensus/execution
client.

Once we have received your bug report, we will try to reproduce it and provide a
more detailed response. When the reported bug has been successfully reproduced,
the team will work on a fix.

## Audits

The upstream codebase this fork derives from was assessed on 2022-06-20 by
[lotusbumi](https://github.com/lotusbumi): see
[docs/audit-20220620.md](docs/audit-20220620.md).
