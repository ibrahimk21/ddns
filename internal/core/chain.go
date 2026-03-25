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

// Resolve returns the current state for a domain.
// BUG: dereferences without checking nil bucket value; fixed in commit 14.
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