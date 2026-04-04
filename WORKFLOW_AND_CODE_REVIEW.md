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

Run unit tests:

~~~powershell
& "C:\Program Files\Go\bin\go.exe" test ./...
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

## 8. Code walkthrough (interview-friendly)

## 8.1 The data model (what is stored)

The core types are in internal/core/types.go:

- Transaction: the signed intent to register or update a domain.
- DomainRecord: the resolved state for a domain (current truth).
- Block: the append-only history entry that wraps a transaction.

Important detail: transactions are signed over a canonical view that excludes
the signature field. That prevents signature self-reference and ensures the
payload is deterministic across implementations.

## 8.2 Cryptography and signing flow

internal/core/crypto.go provides:

- Ed25519 key generation.
- SignHex / VerifyHex for hex-encoded keys and signatures.
- SHA256Hex for block hashing.

Signing flow in practice:
1) CLI builds a Transaction with domain, IP, TTL, owner public key, version.
2) It calls SigningBytes() to get canonical bytes.
3) It signs those bytes with the owner private key.
4) The signature is attached to the Transaction and submitted.

## 8.3 Validation gates (input quality)

internal/core/validation.go blocks bad input early:

- Domain rules (label length, hyphen edges, must have a TLD).
- IPv4 parsing only (A-record only).
- TTL bounds (60..86400).
- Timestamp skew window to avoid far-future or stale payloads.

This keeps malformed or unsafe data from reaching consensus logic.

## 8.4 State machine and persistence

internal/core/chain.go is the core engine:

- BoltDB stores two buckets:
  - blocks: immutable chain history
  - state: current DomainRecord per domain
- SubmitSignedTransaction does all checks and state updates atomically:
  - validates shape and owner signature
  - enforces register vs update rules
  - enforces monotonic versioning (no stale updates)
  - creates block hash and validator signature
  - persists block + updates state in one DB transaction

VerifyImmutability replays the chain to prove:
- prev-hash linkage is intact
- block signatures and transaction signatures are valid
- validators are authorized

## 8.5 Network API surface

internal/node/server.go exposes two interfaces:

- JSON-RPC /rpc:
  - submitTx (state-changing)
  - resolve and status (read-only)
- REST:
  - GET /resolve/{domain}
  - GET /status
  - GET /verify (immutability check)

This separation makes it easy to script demo flows while keeping
state-changing operations explicit through JSON-RPC.

## 8.6 CLI orchestration

cmd/ddns/main.go wires everything together:

- keygen creates owner/validator keys.
- serve starts the node and opens the chain DB.
- register/update build and sign transactions, then POST to /rpc.
- resolve/status/verify call the REST endpoints.
- Key files are normalized to handle UTF-8 BOM and UTF-16 from PowerShell.

## 9. Known limitations of this MVP

- Single-process node behavior, not full peer-to-peer sync yet.
- No domain expiration lifecycle.
- No key rotation workflow.
- No dispute resolution/governance flow.
- A-record only.

These are acceptable for a minimal resume project because the core decentralized DNS + PKI security path is implemented and demonstrable.

## 10. Suggested resume demo script

1. Start server.
2. Register example.com.
3. Resolve and show owner key binding.
4. Update with owner key and show new IP.
5. Attempt attacker update and show rejection.
6. Run verify and show chain integrity passes.

This sequence communicates both functionality and security in under 3 minutes.
