package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"

    "ddns-pki/internal/core"
    "ddns-pki/internal/node"
)

type keyFile struct {
    PublicKey  string `json:"public_key"`
    PrivateKey string `json:"private_key"`
}

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    cmd := os.Args[1]
    var err error

    switch cmd {
    case "keygen":
        err = runKeygen()
    case "serve":
        err = runServe(os.Args[2:])
    case "register":
        err = runSubmit(os.Args[2:], core.TxRegister)
    case "update":
        err = runSubmit(os.Args[2:], core.TxUpdate)
    default:
        printUsage()
        os.Exit(1)
    }

    if err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println("ddns - decentralized DNS + PKI MVP")
    fmt.Println("commands:")
    fmt.Println("  keygen")
    fmt.Println("  serve --addr :8080 --db chain.db --validator-key validator.json --validators <pub1,pub2,pub3>")
    fmt.Println("  register --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 1.2.3.4 --ttl 3600 --version 1")
    fmt.Println("  update --rpc http://localhost:8080/rpc --key owner.json --domain example.com --ip 5.6.7.8 --ttl 3600 --version 2")
}

func runKeygen() error {
    pub, priv, err := core.GenerateKeyPairHex()
    if err != nil {
        return err
    }
    out := keyFile{PublicKey: pub, PrivateKey: priv}
    data, _ := json.MarshalIndent(out, "", "  ")
    fmt.Println(string(data))
    return nil
}

func loadKey(path string) (*keyFile, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var k keyFile
    if err := json.Unmarshal(raw, &k); err != nil {
        return nil, err
    }
    if k.PublicKey == "" || k.PrivateKey == "" {
        return nil, errors.New("invalid key file")
    }
    return &k, nil
}

func runServe(args []string) error {
    fs := flag.NewFlagSet("serve", flag.ContinueOnError)
    addr := fs.String("addr", ":8080", "listen address")
    dbPath := fs.String("db", "chain.db", "path to BoltDB file")
    validatorKeyPath := fs.String("validator-key", "", "validator key JSON")
    validatorsRaw := fs.String("validators", "", "comma-separated validator pub keys")
    if err := fs.Parse(args); err != nil {
        return err
    }

    if *validatorKeyPath == "" {
        return errors.New("--validator-key is required")
    }
    key, err := loadKey(*validatorKeyPath)
    if err != nil {
        return err
    }

    validators := []string{}
    if strings.TrimSpace(*validatorsRaw) != "" {
        validators = strings.Split(*validatorsRaw, ",")
    }
    if len(validators) == 0 {
        validators = append(validators, key.PublicKey)
    }

    chain, err := core.OpenChain(*dbPath, key.PublicKey, key.PrivateKey, validators)
    if err != nil {
        return err
    }
    defer chain.Close()

    srv := node.NewServer(chain)
    fmt.Println("listening on", *addr)
    return http.ListenAndServe(*addr, srv.Routes())
}

func runSubmit(args []string, txType core.TxType) error {
    fs := flag.NewFlagSet(string(txType), flag.ContinueOnError)
    rpcURL := fs.String("rpc", "http://localhost:8080/rpc", "JSON-RPC endpoint")
    keyPath := fs.String("key", "", "owner key JSON")
    domain := fs.String("domain", "", "domain name")
    ip := fs.String("ip", "", "IPv4")
    ttl := fs.Uint("ttl", 3600, "TTL in seconds")
    version := fs.Uint64("version", 0, "domain version")
    if err := fs.Parse(args); err != nil {
        return err
    }

    if *keyPath == "" || *domain == "" || *ip == "" || *version == 0 {
        return errors.New("missing required flags")
    }

    key, err := loadKey(*keyPath)
    if err != nil {
        return err
    }

    tx := core.Transaction{
        Type:      txType,
        Domain:    *domain,
        IP:        *ip,
        TTL:       uint32(*ttl),
        OwnerPub:  key.PublicKey,
        Version:   *version,
        Timestamp: core.NewTimestamp(),
    }
    msg, err := tx.SigningBytes()
    if err != nil {
        return err
    }
    sig, err := core.SignHex(key.PrivateKey, msg)
    if err != nil {
        return err
    }
    tx.Signature = sig

    req := map[string]any{
        "jsonrpc": "2.0",
        "method":  "submitTx",
        "params":  tx,
        "id":      1,
    }

    return postAndPrint(*rpcURL, req)
}

func postAndPrint(url string, payload any) error {
    raw, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    var pretty bytes.Buffer
    if err := json.Indent(&pretty, body, "", "  "); err == nil {
        fmt.Println(pretty.String())
    } else {
        fmt.Println(string(body))
    }
    return nil
}