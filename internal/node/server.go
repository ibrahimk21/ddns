package node

import (
    "encoding/json"
    "net/http"
    "strings"

    "ddns-pki/internal/core"
)

type Server struct {
    chain *core.Chain
}

func NewServer(chain *core.Chain) *Server {
    return &Server{chain: chain}
}

func (s *Server) Routes() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/resolve/", s.handleResolve)
    mux.HandleFunc("/status", s.handleStatus)
    return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
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