package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/evadeplayer/api/internal/handler"
	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/repository"
	"github.com/evadeplayer/api/internal/service"
)

// --- shared in-memory user store (same as service tests) ---

type memUsers struct {
	mu      sync.Mutex
	byEmail map[string]*model.User
	byID    map[string]*model.User
	seq     int
}

func newMemUsers() *memUsers {
	return &memUsers{byEmail: make(map[string]*model.User), byID: make(map[string]*model.User)}
}

func (m *memUsers) Create(_ context.Context, u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[u.Email]; ok {
		return repository.ErrEmailTaken
	}
	m.seq++
	u.ID = fmt.Sprintf("u%d", m.seq)
	u.CreatedAt = time.Now()
	cp := *u
	m.byEmail[u.Email] = &cp
	m.byID[u.ID] = &cp
	return nil
}

func (m *memUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUsers) FindByID(_ context.Context, id string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func newAuthHandler() *handler.AuthHandler {
	svc := service.NewAuthService(newMemUsers(), "test-jwt-secret-32-chars-minimum")
	return handler.NewAuthHandler(svc)
}

func postJSON(h http.HandlerFunc, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// --- Register ---

func TestAuthHandler_Register_Success(t *testing.T) {
	h := newAuthHandler()
	rr := postJSON(h.Register, map[string]string{"email": "a@b.com", "password": "password123"})
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] == nil {
		t.Error("response must contain id")
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	h := newAuthHandler()
	body := map[string]string{"email": "dup@b.com", "password": "password123"}
	postJSON(h.Register, body)
	rr := postJSON(h.Register, body)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	h := newAuthHandler()
	rr := postJSON(h.Register, map[string]string{"email": "x@y.com"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	h := newAuthHandler()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not-json"))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// --- Login ---

func TestAuthHandler_Login_Success(t *testing.T) {
	h := newAuthHandler()
	creds := map[string]string{"email": "l@b.com", "password": "password123"}
	postJSON(h.Register, creds)
	rr := postJSON(h.Login, creds)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("login response must contain token")
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h := newAuthHandler()
	postJSON(h.Register, map[string]string{"email": "wp@b.com", "password": "correct"})
	rr := postJSON(h.Login, map[string]string{"email": "wp@b.com", "password": "wrong"})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthHandler_Login_UnknownEmail(t *testing.T) {
	h := newAuthHandler()
	rr := postJSON(h.Login, map[string]string{"email": "no@b.com", "password": "anything"})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}
