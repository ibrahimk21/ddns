package core

import (
    "crypto/ed25519"
    "crypto/rand"
    "encoding/hex"
)

func GenerateKeyPairHex() (pubHex string, privHex string, err error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return "", "", err
    }
    return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}