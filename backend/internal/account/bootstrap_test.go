package account

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

type fakeBootstrapService struct {
	account *Account
	findErr error
	created *Account
}

func (s *fakeBootstrapService) FindByUsername(context.Context, string) (*Account, error) {
	return s.account, s.findErr
}

func (s *fakeBootstrapService) CreateAccount(_ context.Context, account *Account) error {
	s.created = account
	return nil
}

func TestEnsureBootstrapAccountCreatesMissingAccount(t *testing.T) {
	service := &fakeBootstrapService{findErr: gorm.ErrRecordNotFound}
	created, err := EnsureBootstrapAccount(context.Background(), service, "interviewer", "password123")
	if err != nil {
		t.Fatalf("ensure account: %v", err)
	}
	if !created || service.created == nil || service.created.Username != "interviewer" {
		t.Fatal("expected account creation")
	}
}

func TestEnsureBootstrapAccountIsIdempotent(t *testing.T) {
	service := &fakeBootstrapService{account: &Account{Username: "interviewer"}}
	created, err := EnsureBootstrapAccount(context.Background(), service, "interviewer", "password123")
	if err != nil {
		t.Fatalf("ensure account: %v", err)
	}
	if created || service.created != nil {
		t.Fatal("existing account must not be recreated")
	}
}

func TestEnsureBootstrapAccountValidatesCredentials(t *testing.T) {
	service := &fakeBootstrapService{findErr: errors.New("must not be called")}
	if _, err := EnsureBootstrapAccount(context.Background(), service, "", "password123"); err == nil {
		t.Fatal("expected username error")
	}
	if _, err := EnsureBootstrapAccount(context.Background(), service, "demo", "short"); err == nil {
		t.Fatal("expected password error")
	}
}
