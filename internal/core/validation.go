package core

import (
    "errors"
    "net"
    "regexp"
    "strings"
    "time"
)

var domainLabelRe = regexp.MustCompile(`^[a-zA-Z0-9-]{1,63}$`)

func ValidateDomain(domain string) error {
    if len(domain) == 0 || len(domain) > 253 {
        return errors.New("invalid domain length")
    }
    labels := strings.Split(domain, ".")
    if len(labels) < 2 {
        return errors.New("domain must include a TLD")
    }
    for _, label := range labels {
        if !domainLabelRe.MatchString(label) {
            return errors.New("invalid domain label")
        }
        if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
            return errors.New("domain label cannot start or end with hyphen")
        }
    }
    return nil
}

func ValidateIPv4(ip string) error {
    parsed := net.ParseIP(ip)
    if parsed == nil || parsed.To4() == nil {
        return errors.New("invalid IPv4 address")
    }
    return nil
}

func ValidateTxShape(tx Transaction) error {
    if tx.Type != TxRegister && tx.Type != TxUpdate {
        return errors.New("unsupported transaction type")
    }
    if err := ValidateDomain(tx.Domain); err != nil {
        return err
    }
    if err := ValidateIPv4(tx.IP); err != nil {
        return err
    }
    if tx.TTL < 60 || tx.TTL > 86400 {
        return errors.New("ttl must be between 60 and 86400")
    }
    if tx.OwnerPub == "" || tx.Signature == "" {
        return errors.New("missing owner public key or signature")
    }

    // Keep a small skew window to avoid very stale or far-future signed payloads.
    now := time.Now().Unix()
    if tx.Timestamp < now-600 || tx.Timestamp > now+600 {
        return errors.New("timestamp outside allowed skew")
    }
    return nil
}