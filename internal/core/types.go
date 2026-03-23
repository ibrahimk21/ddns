package core

import (
    "encoding/json"
    "time"
)

type TxType string

const (
    TxRegister TxType = "REGISTER"
    TxUpdate   TxType = "UPDATE"
)

type Transaction struct {
    Type      TxType  `json:"type"`
    Domain    string  `json:"domain"`
    IP        string  `json:"ip"`
    TTL       uint32  `json:"ttl"`
    OwnerPub  string  `json:"owner_pub"`
    Version   uint64  `json:"version"`
    Timestamp int64   `json:"timestamp"`
    Signature string  `json:"signature"`
}

type txSigningView struct {
    Type      TxType `json:"type"`
    Domain    string `json:"domain"`
    IP        string `json:"ip"`
    TTL       uint32 `json:"ttl"`
    OwnerPub  string `json:"owner_pub"`
    Version   uint64 `json:"version"`
    Timestamp int64  `json:"timestamp"`
}

func (t Transaction) SigningBytes() ([]byte, error) {
    view := txSigningView{
        Type:      t.Type,
        Domain:    t.Domain,
        IP:        t.IP,
        TTL:       t.TTL,
        OwnerPub:  t.OwnerPub,
        Version:   t.Version,
        Timestamp: t.Timestamp,
    }
    return json.Marshal(view)
}

type DomainRecord struct {
    Domain       string `json:"domain"`
    IP           string `json:"ip"`
    TTL          uint32 `json:"ttl"`
    OwnerPub     string `json:"owner_pub"`
    Version      uint64 `json:"version"`
    UpdatedAt    int64  `json:"updated_at"`
    LastTxHash   string `json:"last_tx_hash"`
    LastBlockNum uint64 `json:"last_block_num"`
}

type Block struct {
    Height        uint64      `json:"height"`
    PrevHash      string      `json:"prev_hash"`
    Timestamp     int64       `json:"timestamp"`
    ValidatorPub  string      `json:"validator_pub"`
    Tx            Transaction `json:"tx"`
    TxHash        string      `json:"tx_hash"`
    BlockHash     string      `json:"block_hash"`
    BlockSig      string      `json:"block_sig"`
}

func NewTimestamp() int64 {
    return time.Now().Unix()
}