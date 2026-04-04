# Decentralized DNS / PKI MVP (Go)

A minimal resume-ready implementation of decentralized DNS + PKI binding.

## What This MVP Implements

- Append-only blockchain ledger of DNS state transitions.
- A-record support only (domain -> IPv4).
- Public key ownership binding on registration.
- Signed state-changing transactions (`REGISTER`, `UPDATE`).
- Spoofing prevention through signature checks against bound owner key.
- Basic Proof-of-Authority block signing (fixed validator set).
- BoltDB persistence.
- JSON-RPC and REST endpoints.
- CLI for keygen/register/update/resolve/status/verify.

## Project Structure

- `cmd/ddns/main.go` - CLI and node entry point.
- `internal/core/types.go` - protocol data structures.
- `internal/core/crypto.go` - Ed25519 and hashing helpers.
- `internal/core/validation.go` - DNS/IP/transaction shape checks.
- `internal/core/chain.go` - immutable ledger + state transitions + persistence.
- `internal/node/server.go` - JSON-RPC + REST handlers.

## Build

Install Go 1.22+ and run:

```bash
go mod tidy
go build ./cmd/ddns
```

## Testing

Run unit tests (core logic + server handlers):

```bash
go test ./...
```

## Quick Start

1) Generate validator key:

```bash
ddns keygen > validator.json
```

2) Start node:

```bash
ddns serve --addr :8080 --db chain.db --validator-key validator.json
```

3) Generate owner key:

```bash
ddns keygen > owner.json
```

4) Register domain:

```bash
ddns register --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 1.2.3.4 --ttl 3600 --version 1
```

5) Resolve domain:

```bash
ddns resolve --url http://localhost:8080/resolve/example.com
```

6) Update domain (must use same owner key):

```bash
ddns update --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 5.6.7.8 --ttl 3600 --version 2
```

7) Verify chain immutability:

```bash
ddns verify --url http://localhost:8080/verify
```

## API

### JSON-RPC (`POST /rpc`)

- `submitTx` with full signed transaction object.
- `resolve` with `{"domain":"example.com"}`.
- `status` with `{}`.

### REST

- `GET /resolve/{domain}`
- `GET /status`
- `GET /verify`

## Notes

- This is an MVP and intentionally excludes domain expiration, key rotation, and dispute recovery.
- For a multi-node demo, run multiple instances with different DB files and validator keys, then use a shared validator list.