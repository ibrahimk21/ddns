package core

import (
    "encoding/json"
    "errors"
    "fmt"
    "sort"

    bolt "go.etcd.io/bbolt"
)

var (
    blocksBucket  = []byte("blocks")
    stateBucket   = []byte("state")
    metaBucket    = []byte("meta")
    chainTipKey   = []byte("chain_tip")
    genesisTipVal = []byte("0")
)

type Chain struct {
    db            *bolt.DB
    validatorPub  string
    validatorPriv string
    validators    map[string]bool
}

func OpenChain(path string, validatorPub string, validatorPriv string, validators []string) (*Chain, error) {
    db, err := bolt.Open(path, 0600, nil)
    if err != nil {
        return nil, err
    }

    c := &Chain{
        db:            db,
        validatorPub:  validatorPub,
        validatorPriv: validatorPriv,
        validators:    map[string]bool{},
    }
    for _, v := range validators {
        if v != "" {
            c.validators[v] = true
        }
    }
    if len(c.validators) == 0 {
        c.validators[validatorPub] = true
    }

    err = db.Update(func(tx *bolt.Tx) error {
        if _, e := tx.CreateBucketIfNotExists(blocksBucket); e != nil {
            return e
        }
        if _, e := tx.CreateBucketIfNotExists(stateBucket); e != nil {
            return e
        }
        mb, e := tx.CreateBucketIfNotExists(metaBucket)
        if e != nil {
            return e
        }
        if mb.Get(chainTipKey) == nil {
            return mb.Put(chainTipKey, genesisTipVal)
        }
        return nil
    })
    if err != nil {
        _ = db.Close()
        return nil, err
    }

    return c, nil
}

func (c *Chain) Close() error {
    return c.db.Close()
}

func (c *Chain) getTip(tx *bolt.Tx) (uint64, error) {
    raw := tx.Bucket(metaBucket).Get(chainTipKey)
    var tip uint64
    if raw == nil {
        return 0, errors.New("missing chain tip")
    }
    if _, err := fmt.Sscanf(string(raw), "%d", &tip); err != nil {
        return 0, err
    }
    return tip, nil
}

func (c *Chain) setTip(tx *bolt.Tx, tip uint64) error {
    return tx.Bucket(metaBucket).Put(chainTipKey, []byte(fmt.Sprintf("%d", tip)))
}

func (c *Chain) Resolve(domain string) (*DomainRecord, error) {
    var rec DomainRecord
    err := c.db.View(func(tx *bolt.Tx) error {
        raw := tx.Bucket(stateBucket).Get([]byte(domain))
        return json.Unmarshal(raw, &rec)
    })
    if err != nil {
        return nil, err
    }
    return &rec, nil
}

func (c *Chain) Status() (map[string]any, error) {
    out := map[string]any{}
    err := c.db.View(func(tx *bolt.Tx) error {
        tip, err := c.getTip(tx)
        if err != nil {
            return err
        }
        out["height"] = tip
        out["validator_pub"] = c.validatorPub

        vals := make([]string, 0, len(c.validators))
        for v := range c.validators {
            vals = append(vals, v)
        }
        sort.Strings(vals)
        out["validators"] = vals
        return nil
    })
    return out, err
}

func (c *Chain) SubmitSignedTransaction(txIn Transaction) (*Block, error) {
    if err := ValidateTxShape(txIn); err != nil {
        return nil, err
    }

    msg, err := txIn.SigningBytes()
    if err != nil {
        return nil, err
    }
    if !VerifyHex(txIn.OwnerPub, msg, txIn.Signature) {
        return nil, errors.New("invalid transaction signature")
    }

    var committed *Block
    err = c.db.Update(func(dbtx *bolt.Tx) error {
        tip, err := c.getTip(dbtx)
        if err != nil {
            return err
        }

        stateB := dbtx.Bucket(stateBucket)
        rawExisting := stateB.Get([]byte(txIn.Domain))
        var existing *DomainRecord
        if rawExisting != nil {
            var tmp DomainRecord
            if err := json.Unmarshal(rawExisting, &tmp); err != nil {
                return err
            }
            existing = &tmp
        }

        switch txIn.Type {
        case TxRegister:
            if existing != nil {
                return errors.New("domain already registered")
            }
            if txIn.Version != 1 {
                return errors.New("register version must be 1")
            }
        case TxUpdate:
            if existing == nil {
                return errors.New("domain does not exist")
            }
            if existing.OwnerPub != txIn.OwnerPub {
                return errors.New("update signer does not match domain owner")
            }
            // BUG: '<' allows replaying the same version; tightened to '<=' in commit 19.
            if txIn.Version < existing.Version {
                return errors.New("update version must increase")
            }
        default:
            return errors.New("unsupported transaction type")
        }

        prevHash := ""
        if tip > 0 {
            key := []byte(fmt.Sprintf("%d", tip))
            prevRaw := dbtx.Bucket(blocksBucket).Get(key)
            if prevRaw == nil {
                return errors.New("missing previous block")
            }
            var prev Block
            if err := json.Unmarshal(prevRaw, &prev); err != nil {
                return err
            }
            prevHash = prev.BlockHash
        }

        txBytes, err := json.Marshal(txIn)
        if err != nil {
            return err
        }
        txHash := SHA256Hex(txBytes)

        if !c.validators[c.validatorPub] {
            return errors.New("local validator is not in validator set")
        }

        block := Block{
            Height:       tip + 1,
            PrevHash:     prevHash,
            Timestamp:    NewTimestamp(),
            ValidatorPub: c.validatorPub,
            Tx:           txIn,
            TxHash:       txHash,
        }

        // Block hash binds chain linkage + transaction content.
        hashMaterial, err := json.Marshal(struct {
            Height       uint64 `json:"height"`
            PrevHash     string `json:"prev_hash"`
            Timestamp    int64  `json:"timestamp"`
            ValidatorPub string `json:"validator_pub"`
            TxHash       string `json:"tx_hash"`
        }{
            Height:       block.Height,
            PrevHash:     block.PrevHash,
            Timestamp:    block.Timestamp,
            ValidatorPub: block.ValidatorPub,
            TxHash:       block.TxHash,
        })
        if err != nil {
            return err
        }
        block.BlockHash = SHA256Hex(hashMaterial)

        blockSig, err := SignHex(c.validatorPriv, []byte(block.BlockHash))
        if err != nil {
            return err
        }
        block.BlockSig = blockSig

        if !VerifyHex(block.ValidatorPub, []byte(block.BlockHash), block.BlockSig) {
            return errors.New("invalid block signature")
        }

        blockBytes, err := json.Marshal(block)
        if err != nil {
            return err
        }

        if err := dbtx.Bucket(blocksBucket).Put([]byte(fmt.Sprintf("%d", block.Height)), blockBytes); err != nil {
            return err
        }

        next := DomainRecord{
            Domain:       txIn.Domain,
            IP:           txIn.IP,
            TTL:          txIn.TTL,
            OwnerPub:     txIn.OwnerPub,
            Version:      txIn.Version,
            UpdatedAt:    block.Timestamp,
            LastTxHash:   txHash,
            LastBlockNum: block.Height,
        }
        nextBytes, err := json.Marshal(next)
        if err != nil {
            return err
        }
        if err := stateB.Put([]byte(next.Domain), nextBytes); err != nil {
            return err
        }

        if err := c.setTip(dbtx, block.Height); err != nil {
            return err
        }

        committed = &block
        return nil
    })
    if err != nil {
        return nil, err
    }

    return committed, nil
}

func (c *Chain) VerifyImmutability() error {
    return c.db.View(func(tx *bolt.Tx) error {
        tip, err := c.getTip(tx)
        if err != nil {
            return err
        }
        blocks := tx.Bucket(blocksBucket)

        prevHash := ""
        for i := uint64(1); i <= tip; i++ {
            raw := blocks.Get([]byte(fmt.Sprintf("%d", i)))
            if raw == nil {
                return fmt.Errorf("missing block at height %d", i)
            }
            var b Block
            if err := json.Unmarshal(raw, &b); err != nil {
                return err
            }
            if b.PrevHash != prevHash {
                return fmt.Errorf("broken prev hash linkage at height %d", i)
            }
            if !c.validators[b.ValidatorPub] {
                return fmt.Errorf("unauthorized validator at height %d", i)
            }
            if !VerifyHex(b.ValidatorPub, []byte(b.BlockHash), b.BlockSig) {
                return fmt.Errorf("invalid block signature at height %d", i)
            }

            hashMaterial, err := json.Marshal(struct {
                Height       uint64 `json:"height"`
                PrevHash     string `json:"prev_hash"`
                Timestamp    int64  `json:"timestamp"`
                ValidatorPub string `json:"validator_pub"`
                TxHash       string `json:"tx_hash"`
            }{
                Height:       b.Height,
                PrevHash:     b.PrevHash,
                Timestamp:    b.Timestamp,
                ValidatorPub: b.ValidatorPub,
                TxHash:       b.TxHash,
            })
            if err != nil {
                return err
            }
            recalculated := SHA256Hex(hashMaterial)
            if b.BlockHash != recalculated {
                return fmt.Errorf("block hash mismatch at height %d", i)
            }

            txMsg, err := b.Tx.SigningBytes()
            if err != nil {
                return err
            }
            if !VerifyHex(b.Tx.OwnerPub, txMsg, b.Tx.Signature) {
                return fmt.Errorf("transaction signature mismatch at height %d", i)
            }

            prevHash = b.BlockHash
        }
        return nil
    })
}