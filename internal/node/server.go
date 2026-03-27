package node

import (
    "encoding/json"
    "io"
    "net/http"
    "strings"

    "ddns-pki/internal/core"
)

type Server struct {
    chain *core.Chain
}

type rpcRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
    ID      any             `json:"id"`
}

type rpcResponse struct {
    JSONRPC string `json:"jsonrpc"`
    Result  any    `json:"result,omitempty"`
    Error   any    `json:"error,omitempty"`
    ID      any    `json:"id"`
}

func NewServer(chain *core.Chain) *Server {
    return &Server{chain: chain}
}

func (s *Server) Routes() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/rpc", s.handleRPC)
    mux.HandleFunc("/resolve/", s.handleResolve)
    mux.HandleFunc("/status", s.handleStatus)
    return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
        return
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    var req rpcRequest
    if err := json.Unmarshal(body, &req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }

    resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}

    switch req.Method {
    case "submitTx":
        var tx core.Transaction
        if err := json.Unmarshal(req.Params, &tx); err != nil {
            resp.Error = err.Error()
            writeJSON(w, http.StatusBadRequest, resp)
            return
        }
        block, err := s.chain.SubmitSignedTransaction(tx)
        if err != nil {
            resp.Error = err.Error()
        } else {
            resp.Result = block
        }
    case "resolve":
        var p struct {
            Domain string `json:"domain"`
        }
        if err := json.Unmarshal(req.Params, &p); err != nil {
            resp.Error = err.Error()
            writeJSON(w, http.StatusBadRequest, resp)
            return
        }
        rec, err := s.chain.Resolve(p.Domain)
        if err != nil {
            resp.Error = err.Error()
        } else {
            resp.Result = rec
        }
    case "status":
        st, err := s.chain.Status()
        if err != nil {
            resp.Error = err.Error()
        } else {
            resp.Result = st
        }
    default:
        resp.Error = "method not found"
    }

    writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
        return
    }
    domain := strings.TrimPrefix(r.URL.Path, "/resolve/")
    rec, err := s.chain.Resolve(domain)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    if rec == nil {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
        return
    }
    writeJSON(w, http.StatusOK, rec)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
        return
    }
    st, err := s.chain.Status()
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    writeJSON(w, http.StatusOK, st)
}