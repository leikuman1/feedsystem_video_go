package account

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type BootstrapAccountService interface {
	FindByUsername(ctx context.Context, username string) (*Account, error)
	CreateAccount(ctx context.Context, account *Account) error
}

func EnsureBootstrapAccount(
	ctx context.Context,
	service BootstrapAccountService,
	username, password string,
) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" && password == "" {
		return false, nil
	}
	if username == "" {
		return false, errors.New("bootstrap username is required")
	}
	if len(password) < 8 {
		return false, errors.New("bootstrap password must contain at least 8 characters")
	}

	if _, err := service.FindByUsername(ctx, username); err == nil {
		return false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, fmt.Errorf("find bootstrap account: %w", err)
	}

	if err := service.CreateAccount(ctx, &Account{Username: username, Password: password}); err != nil {
		return false, fmt.Errorf("create bootstrap account: %w", err)
	}
	return true, nil
}
