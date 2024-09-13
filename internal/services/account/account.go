//go:generate mockgen -source=account.go -destination=./mock/account.go -package=mock
package account

import (
	"context"
	"errors"
	"fmt"
	"time"
	"tldw/config"
	"tldw/logging"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"tldw/internal/errorx"
	"tldw/internal/model"
)

type Repository interface {
	Create(ctx context.Context, account *model.Account) (*model.Account, error)
	GetByUsername(ctx context.Context, username string) (*model.Account, error)
	ChangePassword(ctx context.Context, id string, oldPassword string, newPassword string) error
}

type Service struct {
	cfg  config.Auth
	repo Repository
}

func NewService(cfg config.Auth, r Repository) Service {
	return Service{cfg: cfg, repo: r}
}

func (s Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	a, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		logging.Get().Error("failed to get user by username", "username", req.Username, "error", err)
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(req.Password))
	if err != nil {
		logging.Get().Error("invalid credentials", "username", req.Username, "error", err)
		return nil, errorx.NewError(
			errors.New("invalid credentials"),
			errorx.ErrUnauthorized,
		)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"email": a.Username,
		"sub":   a.Id,
		"exp":   now.Add(time.Hour * s.cfg.TokenExp).Unix(),
		"iat":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, errorx.NewError(err, errorx.ErrInternal)
	}

	res := new(LoginResponse)
	res.Token = signedToken

	return res, nil
}

func (s Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errorx.NewError(
			fmt.Errorf("failed to hash password: %w", err),
			errorx.ErrInternal,
		)
	}

	a := model.NewAccount().
		WithUsername(req.Username).
		WithPassword(string(hashPwd))

	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return nil, errorx.NewError(
			fmt.Errorf("failed to create account: %w", err),
			errorx.ErrInternal,
		)
	}

	res := new(RegisterResponse)
	res.AccountId = created.Id

	return res, nil
}

func (s Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) (*ChangePasswordResponse, error) {
	err := s.repo.ChangePassword(ctx, req.AccountId, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}

	res := new(ChangePasswordResponse)
	res.AccountId = req.AccountId

	return res, nil
}
