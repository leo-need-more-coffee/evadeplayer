package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/repository"
	"github.com/evadeplayer/api/internal/service"
)

// --- in-memory stub ---

type memUserStore struct {
	mu      sync.Mutex
	byEmail map[string]*model.User
	byID    map[string]*model.User
	seq     int
}

func newMemUserStore() *memUserStore {
	return &memUserStore{
		byEmail: make(map[string]*model.User),
		byID:    make(map[string]*model.User),
	}
}

func (m *memUserStore) Create(_ context.Context, u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[u.Email]; exists {
		return repository.ErrEmailTaken
	}
	m.seq++
	u.ID = fmt.Sprintf("uid-%d", m.seq)
	u.CreatedAt = time.Now()
	cp := *u
	m.byEmail[u.Email] = &cp
	m.byID[u.ID] = &cp
	return nil
}

func (m *memUserStore) FindByEmail(_ context.Context, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUserStore) FindByID(_ context.Context, id string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

const testJWTSecret = "super-secret-jwt-key-32-chars-min"

func newAuthSvc() *service.AuthService {
	return service.NewAuthService(newMemUserStore(), testJWTSecret)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	svc := newAuthSvc()
	u, err := svc.Register(context.Background(), "a@b.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Error("user ID must be set after register")
	}
	if u.Password != "" {
		t.Error("Password field must be empty in returned user")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newAuthSvc()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "dup@b.com", "password123")
	_, err := svc.Register(ctx, "dup@b.com", "password456")
	if !errors.Is(err, repository.ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := newAuthSvc()
	_, err := svc.Register(context.Background(), "x@y.com", "short")
	if err == nil {
		t.Error("expected error for short password")
	}
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	svc := newAuthSvc()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "login@b.com", "correct-password")
	token, err := svc.Login(ctx, "login@b.com", "correct-password")
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newAuthSvc()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "wp@b.com", "correct-password")
	_, err := svc.Login(ctx, "wp@b.com", "wrong-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := newAuthSvc()
	_, err := svc.Login(context.Background(), "nobody@b.com", "anything")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// --- ValidateToken ---

func TestValidateToken_Valid(t *testing.T) {
	svc := newAuthSvc()
	ctx := context.Background()
	_, _ = svc.Register(ctx, "vt@b.com", "password123")
	token, _ := svc.Login(ctx, "vt@b.com", "password123")

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if userID == "" {
		t.Error("expected non-empty userID from token")
	}
}

func TestValidateToken_Garbage(t *testing.T) {
	svc := newAuthSvc()
	_, err := svc.ValidateToken("not.a.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := service.NewAuthService(newMemUserStore(), "secret-one-32-chars-xxxxxxxxxx1")
	svc2 := service.NewAuthService(newMemUserStore(), "secret-two-32-chars-xxxxxxxxxx2")

	ctx := context.Background()
	_, _ = svc1.Register(ctx, "a@b.com", "password123")
	token, _ := svc1.Login(ctx, "a@b.com", "password123")

	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}
