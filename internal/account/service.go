//go:generate mockgen -source=service.go -destination=./mock/service.go -package=mock
package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/errorx"
)

type Repository interface {
	Create(ctx context.Context, a *Account) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Identity, error)
	ChangePassword(ctx context.Context, id string, oldPassword string, newPassword string) error
}

type Service struct {
	cfg  config.AuthConfig
	repo Repository
}

func NewService(cfg config.AuthConfig, r Repository) Service {
	return Service{cfg: cfg, repo: r}
}

func (s Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	logger := slogging.Slogger()

	a, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get user by username", "email", req.Email, "error", err)
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(req.Password))
	if err != nil {
		logger.ErrorContext(ctx, "invalid credentials", "email", req.Email, "error", err)
		return nil, errorx.NewError(
			errors.New("invalid credentials"),
			errorx.ErrUnauthorized,
		)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"email": a.Email,
		"sub":   a.AccountId,
		"exp":   now.Add(time.Hour * s.cfg.TokenExp).Unix(),
		"iat":   now.Unix(),
		"roles": a.Roles,
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
		slogging.Slogger().ErrorContext(ctx, "failed to hash password", "error", err)
		return nil, errorx.NewError(
			fmt.Errorf("failed to hash password: %w", err),
			errorx.ErrInternal,
		)
	}

	a := New().WithEmail(req.Email).WithPassword(string(hashPwd))

	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return nil, err
	}

	res := new(RegisterResponse)
	res.AccountId = created.Id

	return res, nil
}

func (s Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) (*ChangePasswordResponse, error) {
	err := s.repo.ChangePassword(ctx, req.AccountId, req.OldPassword, req.NewPassword)
	if err != nil {
		slogging.Slogger().ErrorContext(ctx, "failed to change password", "error", err)
		return nil, err
	}

	res := new(ChangePasswordResponse)
	res.AccountId = req.AccountId

	return res, nil
}
