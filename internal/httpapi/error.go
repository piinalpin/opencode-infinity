package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"opencode-infinity/internal/gateway"
)

func newRequestID() string {
	var id [16]byte
	_, _ = rand.Read(id[:])
	return "req_" + hex.EncodeToString(id[:])
}
func writeModelError(w http.ResponseWriter, id string, f *gateway.Failure) {
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("X-Infinity-Error-Source", "gateway")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(f.Status)
	payload := struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}{RequestID: id}
	payload.Error.Message = "Request tidak dapat diproses."
	payload.Error.Type = "gateway_error"
	payload.Error.Code = f.Code
	_ = json.NewEncoder(w).Encode(payload)
}
