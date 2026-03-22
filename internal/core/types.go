package core

type TxType string

const (
    TxRegister TxType = "REGISTER"
    TxUpdate   TxType = "UPDATE"
)

type Transaction struct {
    Type      TxType `json:"type"`
    Domain    string `json:"domain"`
    IP        string `json:"ip"`
    TTL       uint32 `json:"ttl"`
    OwnerPub  string `json:"owner_pub"`
    Version   uint64 `json:"version"`
    Timestamp int64  `json:"timestamp"`
    Signature string `json:"signature"`
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