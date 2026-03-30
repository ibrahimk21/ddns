# Decentralized DNS + PKI MVP

## 1. What this project is

This project is a minimal decentralized DNS + PKI system in Go.

It provides:
- An append-only blockchain-like ledger for DNS state changes.
- A-record support only (domain -> IPv4).
- Public key ownership binding at registration time.
- Signed state-changing transactions.
- Authorization checks that block spoofed updates.
- Immutability verification over stored blocks.

## 2. End-to-end workflow summary

The full lifecycle is:
1. Generate keys.
2. Start a validator node.
3. Register a domain with owner signature.
4. Resolve the domain from state.
5. Update domain using same owner key.
6. Verify chain integrity.
7. Run failure tests (spoofed key, stale version, duplicate register).

## 3. Prerequisites

- Windows PowerShell.
- Go installed at C:\Program Files\Go\bin\go.exe.

## 4. Build and run from scratch

From project root:

~~~powershell
& "C:\Program Files\Go\bin\go.exe" mod tidy
& "C:\Program Files\Go\bin\go.exe" build -o ddns.exe ./cmd/ddns
~~~

Generate keys in UTF-8 safely:

~~~powershell
./ddns.exe keygen | Out-File -FilePath validator.json -Encoding utf8
./ddns.exe keygen | Out-File -FilePath owner.json -Encoding utf8
~~~

Start node:

~~~powershell
./ddns.exe serve --addr :8080 --db chain.db --validator-key validator.json
~~~

Expected output:

~~~text
listening on :8080
~~~

Use a second terminal for client commands.

## 5. Happy-path demo with expected behavior

### 5.1 Register

~~~powershell
./ddns.exe register --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 1.2.3.4 --ttl 3600 --version 1
~~~

Expected behavior:
- Returns JSON-RPC result.
- Creates block at height 1.
- Binds example.com ownership to owner_pub.

Observed sample output (real run):

~~~json
{
  "jsonrpc": "2.0",
  "result": {
    "height": 1,
    "prev_hash": "",
    "tx": {
      "type": "REGISTER",
      "domain": "example.com",
      "ip": "1.2.3.4",
      "version": 1
    }
  },
  "id": 1
}
~~~

### 5.2 Resolve

~~~powershell
./ddns.exe resolve --url http://localhost:8080/resolve/example.com
~~~

Expected behavior:
- Returns state record with IP 1.2.3.4 and version 1.

Observed sample output:

~~~json
{
  "domain": "example.com",
  "ip": "1.2.3.4",
  "version": 1,
  "last_block_num": 1
}
~~~

### 5.3 Update with owner key

~~~powershell
./ddns.exe update --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 5.6.7.8 --ttl 3600 --version 2
~~~

Expected behavior:
- Accepted only if signed by bound owner key.
- Creates next block with linked prev_hash.

Observed sample output:

~~~json
{
  "jsonrpc": "2.0",
  "result": {
    "height": 2,
    "prev_hash": "6d5ac1ab1d1638d7947aa8e9ff3f96e663baced2fe2e190ea4040256d5409247",
    "tx": {
      "type": "UPDATE",
      "domain": "example.com",
      "ip": "5.6.7.8",
      "version": 2
    }
  },
  "id": 1
}
~~~

### 5.4 Resolve again

~~~powershell
./ddns.exe resolve --url http://localhost:8080/resolve/example.com
~~~

Expected behavior:
- Returns updated IP 5.6.7.8.
- Version is now 2.

### 5.5 Verify immutability

~~~powershell
./ddns.exe verify --url http://localhost:8080/verify
~~~

Observed output:

~~~json
{
  "ok": true
}
~~~

Meaning:
- Hash-link consistency passed.
- Validator authorization checks passed.
- Block signatures passed.
- Transaction signatures passed.

## 6. Failure scenarios you should test

### 6.1 Spoofed update by attacker key

Command:

~~~powershell
./ddns.exe keygen | Out-File -FilePath attacker.json -Encoding utf8
./ddns.exe update --rpc http://localhost:8080/rpc --key attacker.json --domain example.com --ip 9.9.9.9 --ttl 3600 --version 3
~~~

Observed output:

~~~json
{
  "jsonrpc": "2.0",
  "error": "update signer does not match domain owner",
  "id": 1
}
~~~

What this proves:
- Ownership is key-bound.
- Unauthorized keys cannot modify existing domain state.

### 6.2 Replay/stale version update

Command:

~~~powershell
./ddns.exe update --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 5.6.7.9 --ttl 3600 --version 2
~~~

Observed output:

~~~json
{
  "jsonrpc": "2.0",
  "error": "update version must increase",
  "id": 1
}
~~~

What this proves:
- Monotonic versioning blocks stale/replayed updates.

### 6.3 Duplicate register attempt

Command:

~~~powershell
./ddns.exe register --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 2.2.2.2 --ttl 3600 --version 1
~~~

Observed output:

~~~json
{
  "jsonrpc": "2.0",
  "error": "domain already registered",
  "id": 1
}
~~~

What this proves:
- First valid registration wins.
- Name cannot be re-registered while it exists.

### 6.4 Confirm state did not change after failed attacks

Command:

~~~powershell
./ddns.exe resolve --url http://localhost:8080/resolve/example.com
~~~

Observed output (after failed attacks):

~~~json
{
  "domain": "example.com",
  "ip": "5.6.7.8",
  "version": 2,
  "last_block_num": 2
}
~~~

What this proves:
- Rejected operations are not applied to state.

## 7. How to know it works

You have a valid implementation if all of these are true:
1. Register succeeds once with version 1.
2. Resolve returns owner key and latest A record.
3. Valid owner update with higher version succeeds.
4. Attacker update fails.
5. Stale version update fails.
6. Duplicate registration fails.
7. Verify endpoint returns ok true.

## 8. Code review summary by file

## 8.1 cmd/ddns/main.go

Purpose:
- CLI entrypoint and command routing.

Key behavior:
- Parses commands: keygen, serve, register, update, resolve, status, verify.
- Reads key files and normalizes encoding (UTF-8 BOM + UTF-16 LE/BE supported).
- Creates signed transactions locally before submission.
- Sends JSON-RPC requests and prints pretty JSON responses.

Why relevant:
- This is user-facing orchestration.
- The UTF-16 normalization fixes common PowerShell file encoding issues.

## 8.2 internal/core/types.go

Purpose:
- Canonical protocol models.

Key behavior:
- Defines Transaction, DomainRecord, Block.
- Defines transaction signing view that excludes signature field.
- Provides deterministic signing payload creation.

Why relevant:
- Signing the canonical payload is the foundation for authorization checks.

## 8.3 internal/core/crypto.go

Purpose:
- Cryptographic primitives.

Key behavior:
- Generates Ed25519 keypairs.
- Signs bytes with private key.
- Verifies signatures with public key.
- Hashes data via SHA-256.

Why relevant:
- All PKI ownership and anti-spoofing guarantees depend on this.

## 8.4 internal/core/validation.go

Purpose:
- Input and transaction shape validation.

Key behavior:
- Validates domain syntax and label rules.
- Validates IPv4 format.
- Enforces TTL bounds and timestamp skew window.
- Restricts transaction type to REGISTER and UPDATE.

Why relevant:
- Prevents malformed or stale payloads from entering consensus logic.

## 8.5 internal/core/chain.go

Purpose:
- Ledger state machine + persistence + immutability verification.

Key behavior:
- Initializes BoltDB buckets for blocks, state, metadata.
- Maintains chain tip.
- Applies signed transactions atomically in DB write transaction.
- Enforces registration/update authorization and version rules.
- Creates block hash and validator block signature.
- Persists block and resulting domain state.
- Verifies full chain linkage and signatures in VerifyImmutability.

Why relevant:
- This file is the core security and correctness engine.

## 8.6 internal/node/server.go

Purpose:
- Network API surface.

Key behavior:
- Exposes JSON-RPC endpoint for submitTx/resolve/status.
- Exposes REST endpoints: resolve, status, verify.
- Maps API calls to chain operations and returns JSON responses.

Why relevant:
- This is how external clients interact with ledger and state.

## 9. Architecture notes in plain terms

- The state is current truth per domain in the state bucket.
- The blocks bucket is historical evidence of all accepted updates.
- A valid update must pass both:
  - owner signature verification, and
  - ownership/version state rules.
- Immutability check replays trust assumptions from stored data:
  - prev hash linkage,
  - authorized validator,
  - valid validator signature,
  - valid transaction signature.

## 10. Known limitations of this MVP

- Single-process node behavior, not full peer-to-peer sync yet.
- No domain expiration lifecycle.
- No key rotation workflow.
- No dispute resolution/governance flow.
- A-record only.

These are acceptable for a minimal resume project because the core decentralized DNS + PKI security path is implemented and demonstrable.

## 11. Suggested resume demo script

1. Start server.
2. Register example.com.
3. Resolve and show owner key binding.
4. Update with owner key and show new IP.
5. Attempt attacker update and show rejection.
6. Run verify and show chain integrity passes.

This sequence communicates both functionality and security in under 3 minutes.
