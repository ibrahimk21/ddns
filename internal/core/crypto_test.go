package core

import "testing"

func TestSignAndVerifyHex(t *testing.T) {
    pub, priv, err := GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex: %v", err)
    }

    msg := []byte("hello-world")
    sig, err := SignHex(priv, msg)
    if err != nil {
        t.Fatalf("SignHex: %v", err)
    }

    if !VerifyHex(pub, msg, sig) {
        t.Fatal("VerifyHex returned false for valid signature")
    }
}

func TestVerifyHexRejectsBadSignature(t *testing.T) {
    pub, priv, err := GenerateKeyPairHex()
    if err != nil {
        t.Fatalf("GenerateKeyPairHex: %v", err)
    }

    sig, err := SignHex(priv, []byte("payload"))
    if err != nil {
        t.Fatalf("SignHex: %v", err)
    }

    if VerifyHex(pub, []byte("different"), sig) {
        t.Fatal("VerifyHex accepted mismatched payload")
    }
}
