package main

import (
    "encoding/json"
    "errors"
    "flag"
    "fmt"
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