package core

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "errors"
)

func GenerateKeyPairHex() (pubHex string, privHex string, err error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return "", "", err
    }
    return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}

func SignHex(privHex string, msg []byte) (string, error) {
    priv, err := hex.DecodeString(privHex)
    if err != nil {
        return "", err
    }
    if len(priv) != ed25519.PrivateKeySize {
        return "", errors.New("invalid private key length")
    }
    sig := ed25519.Sign(ed25519.PrivateKey(priv), msg)
    return hex.EncodeToString(sig), nil
}

func VerifyHex(pubHex string, msg []byte, sigHex string) bool {
    pub, err := hex.DecodeString(pubHex)
    if err != nil {
        return false
    }
    sig, err := hex.DecodeString(sigHex)
    if err != nil {
        return false
    }
    if len(pub) != ed25519.PublicKeySize || len(sig) != ed25519.SignatureSize {
        return false
    }
    return ed25519.Verify(ed25519.PublicKey(pub), msg, sig)
}

func SHA256Hex(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}