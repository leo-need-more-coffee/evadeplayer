package handler

import (
	"crypto/hmac"
	"net/http"
	"strconv"
	"time"

	"github.com/evadeplayer/api/internal/service"
)

type HLSAuthHandler struct {
	secret []byte
}

func NewHLSAuthHandler(secret string) *HLSAuthHandler {
	return &HLSAuthHandler{secret: []byte(secret)}
}

func (h *HLSAuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	videoID := q.Get("video_id")
	token := q.Get("token")
	expiresStr := q.Get("expires")

	if videoID == "" || token == "" || expiresStr == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || time.Now().Unix() > expires {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expected := service.ComputeHLSToken(h.secret, videoID, expiresStr)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}
