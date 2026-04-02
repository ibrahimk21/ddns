package core

import (
    "testing"
    "time"
)

func TestValidateDomain(t *testing.T) {
    good := []string{"example.com", "sub.example.com", "a-b.example.co"}
    for _, d := range good {
        if err := ValidateDomain(d); err != nil {
            t.Fatalf("expected valid domain %q: %v", d, err)
        }
    }

    bad := []string{"", "localhost", "-bad.com", "bad-.com", "bad..com"}
    for _, d := range bad {
        if err := ValidateDomain(d); err == nil {
            t.Fatalf("expected invalid domain %q", d)
        }
    }
}

func TestValidateIPv4(t *testing.T) {
    if err := ValidateIPv4("1.2.3.4"); err != nil {
        t.Fatalf("expected valid ip: %v", err)
    }
    if err := ValidateIPv4("999.1.1.1"); err == nil {
        t.Fatal("expected invalid ip")
    }
}

func TestValidateTxShape(t *testing.T) {
    tx := Transaction{
        Type:      TxRegister,
        Domain:    "example.com",
        IP:        "1.2.3.4",
        TTL:       300,
        OwnerPub:  "pub",
        Signature: "sig",
        Timestamp: time.Now().Unix(),
    }

    if err := ValidateTxShape(tx); err != nil {
        t.Fatalf("expected valid tx: %v", err)
    }

    tx.TTL = 10
    if err := ValidateTxShape(tx); err == nil {
        t.Fatal("expected ttl bounds error")
    }
}
