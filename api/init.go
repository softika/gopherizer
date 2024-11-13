package api

import (
	"github.com/go-playground/validator/v10"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"

	// core services
	"github.com/softika/gopherizer/internal/account"
	"github.com/softika/gopherizer/internal/health"
	"github.com/softika/gopherizer/internal/profile"

	// http handler mappers
	"github.com/softika/gopherizer/api/mappers"

	// repositories
	repoAccount "github.com/softika/gopherizer/database/repositories/account"
	healthRepo "github.com/softika/gopherizer/database/repositories/health"
	repoProfile "github.com/softika/gopherizer/database/repositories/profile"
)

type repositories struct {
	health  health.Repository
	account account.Repository
	profile profile.Repository
}

func (r *Router) initRepositories(cfg config.DatabaseConfig) repositories {
	db := database.New(cfg)
	return repositories{
		health:  healthRepo.NewRepository(db),
		account: repoAccount.NewRepository(db),
		profile: repoProfile.NewRepository(db),
	}
}

type services struct {
	account account.Service
	health  health.Service
	profile profile.Service
}

func (r *Router) initServices(cfg config.AuthConfig, s repositories) services {
	return services{
		account: account.NewService(cfg, s.account),
		health:  health.NewService(s.health),
		profile: profile.NewService(s.profile),
	}
}

type handlers struct {
	health Handler[health.Request, *health.Response]

	accountLogin          Handler[account.LoginRequest, *account.LoginResponse]
	accountRegister       Handler[account.RegisterRequest, *account.RegisterResponse]
	accountChangePassword Handler[account.ChangePasswordRequest, *account.ChangePasswordResponse]

	profileCreate Handler[profile.CreateRequest, *profile.Response]
	profileGet    Handler[string, *profile.Response]
	profileUpdate Handler[profile.UpdateRequest, *profile.Response]
	profileDelete Handler[string, bool]
}

func (r *Router) initHandlers(s services) handlers {
	vld := validator.New()

	healthHandler := NewHandler(
		mappers.HealthRequest{},
		mappers.HealthResponse{},
		s.health.Check,
		vld,
	)

	accountLoginHandler := NewHandler(
		mappers.LoginRequest{},
		mappers.LoginResponse{},
		s.account.Login,
		vld,
	)

	accountRegisterHandler := NewHandler(
		mappers.RegisterRequest{},
		mappers.RegisterResponse{},
		s.account.Register,
		vld,
	)

	accountChangePasswordHandler := NewHandler(
		mappers.ChangePasswordRequest{},
		mappers.ChangePasswordResponse{},
		s.account.ChangePassword,
		vld,
	)

	profileCreateHandler := NewHandler(
		mappers.CreateProfileRequest{},
		mappers.CreateProfileResponse{},
		s.profile.Create,
		vld,
	)

	profileGetHandler := NewHandler(
		mappers.GetProfileByIdRequest{},
		mappers.GetProfileResponse{},
		s.profile.GetById,
		vld,
	)

	profileUpdateHandler := NewHandler(
		mappers.UpdateProfileRequest{},
		mappers.UpdateProfileResponse{},
		s.profile.Update,
		vld,
	)

	profileDeleteHandler := NewHandler(
		mappers.DeleteProfileRequest{},
		mappers.DeleteProfileResponse{},
		s.profile.DeleteById,
		vld,
	)

	return handlers{
		health: healthHandler,

		accountRegister:       accountRegisterHandler,
		accountLogin:          accountLoginHandler,
		accountChangePassword: accountChangePasswordHandler,

		profileCreate: profileCreateHandler,
		profileGet:    profileGetHandler,
		profileUpdate: profileUpdateHandler,
		profileDelete: profileDeleteHandler,
	}
}
