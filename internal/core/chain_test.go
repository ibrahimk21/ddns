package core

import (
    "os"
    "path/filepath"
    "testing"
)

func signTx(t *testing.T, priv string, tx Transaction) Transaction {
    t.Helper()
    msg, err := tx.SigningBytes()
    if err != nil {
        t.Fatalf("SigningBytes: %v", err)
    }
    sig, err := SignHex(priv, msg)
    if err != nil {
        t.Fatalf("SignHex: %v", err)
    }
    tx.Signature = sig
    return tx
}

func TestChainRegisterUpdateAndResolve(t *testing.T) {
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "chain.db")

    valPub, valPriv, err := GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex validator: %v", err)
    }
    ownerPub, ownerPriv, err := GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex owner: %v", err)
    }

    chain, err := OpenChain(dbPath, valPub, valPriv, []string{valPub})
    if err != nil {
        t.Fatalf("OpenChain: %v", err)
    }
    t.Cleanup(func() { _ = chain.Close() })

    if rec, err := chain.Resolve("example.com"); err != nil || rec != nil {
        t.Fatalf("expected empty resolve, got rec=%v err=%v", rec, err)
    }

    reg := Transaction{
        Type:      TxRegister,
        Domain:    "example.com",
        IP:        "1.2.3.4",
        TTL:       3600,
        OwnerPub:  ownerPub,
        Version:   1,
        Timestamp: NewTimestamp(),
    }
    reg = signTx(t, ownerPriv, reg)

    if _, err := chain.SubmitSignedTransaction(reg); err != nil {
        t.Fatalf("SubmitSignedTransaction register: %v", err)
    }

    update := Transaction{
        Type:      TxUpdate,
        Domain:    "example.com",
        IP:        "5.6.7.8",
        TTL:       3600,
        OwnerPub:  ownerPub,
        Version:   2,
        Timestamp: NewTimestamp(),
    }
    update = signTx(t, ownerPriv, update)

    if _, err := chain.SubmitSignedTransaction(update); err != nil {
        t.Fatalf("SubmitSignedTransaction update: %v", err)
    }

    rec, err := chain.Resolve("example.com")
    if err != nil {
        t.Fatalf("Resolve: %v", err)
    }
    if rec == nil || rec.IP != "5.6.7.8" || rec.Version != 2 {
        t.Fatalf("unexpected state: %+v", rec)
    }

    if err := chain.VerifyImmutability(); err != nil {
        t.Fatalf("VerifyImmutability: %v", err)
    }

    stale := update
    stale.Version = 2
    stale = signTx(t, ownerPriv, stale)
    if _, err := chain.SubmitSignedTransaction(stale); err == nil {
        t.Fatal("expected stale update rejection")
    }

    dup := reg
    dup = signTx(t, ownerPriv, dup)
    if _, err := chain.SubmitSignedTransaction(dup); err == nil {
        t.Fatal("expected duplicate registration rejection")
    }

    // Ensure db file exists after writes.
    if _, err := os.Stat(dbPath); err != nil {
        t.Fatalf("expected db file: %v", err)
    }
}
