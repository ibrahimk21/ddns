package core

import (
    "crypto/ed25519"
    "crypto/rand"
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