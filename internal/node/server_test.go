package node

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"

    "ddns-pki/internal/core"
)

func TestServerRPCAndResolve(t *testing.T) {
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "chain.db")

    valPub, valPriv, err := core.GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex validator: %v", err)
    }
    ownerPub, ownerPriv, err := core.GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex owner: %v", err)
    }

    chain, err := core.OpenChain(dbPath, valPub, valPriv, []string{valPub})
    if err != nil {
        t.Fatalf("OpenChain: %v", err)
    }
    t.Cleanup(func() { _ = chain.Close() })

    srv := NewServer(chain)

    tx := core.Transaction{
        Type:      core.TxRegister,
        Domain:    "example.com",
        IP:        "1.2.3.4",
        TTL:       3600,
        OwnerPub:  ownerPub,
        Version:   1,
        Timestamp: core.NewTimestamp(),
    }
    msg, err := tx.SigningBytes()
    if err != nil {
        t.Fatalf("SigningBytes: %v", err)
    }
    sig, err := core.SignHex(ownerPriv, msg)
    if err != nil {
        t.Fatalf("SignHex: %v", err)
    }
    tx.Signature = sig

    reqPayload := map[string]any{
        "jsonrpc": "2.0",
        "method":  "submitTx",
        "params":  tx,
        "id":      1,
    }
    raw, _ := json.Marshal(reqPayload)

    req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(raw))
    rec := httptest.NewRecorder()
    srv.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("submitTx status: %d", rec.Code)
    }

    resolveReq := httptest.NewRequest(http.MethodGet, "/resolve/example.com", nil)
    resolveRec := httptest.NewRecorder()
    srv.Routes().ServeHTTP(resolveRec, resolveReq)

    if resolveRec.Code != http.StatusOK {
        t.Fatalf("resolve status: %d", resolveRec.Code)
    }
}
