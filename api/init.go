package api

import (
	"github.com/go-playground/validator/v10"

	"tldw/config"
	"tldw/database"

	// internal
	"tldw/internal/account"
	"tldw/internal/health"
	"tldw/internal/profile"

	// api
	mapAccount "tldw/api/account"
	mapHealth "tldw/api/health"
	mapProfile "tldw/api/profile"

	// repo
	repoAccount "tldw/database/repositories/account"
	repoProfile "tldw/database/repositories/profile"
)

type repositories struct {
	db      database.Service
	account account.Repository
	profile profile.Repository
}

func (r *Router) initRepositories(cfg config.DatabaseConfig) repositories {
	db := database.New(cfg)
	return repositories{
		db:      db,
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
		health:  health.NewService(s.db),
		profile: profile.NewService(s.profile),
	}
}

type handlers struct {
	health Handler[health.Request, *health.Response]

	accountRegister       Handler[account.RegisterRequest, *account.RegisterResponse]
	accountLogin          Handler[account.LoginRequest, *account.LoginResponse]
	accountChangePassword Handler[account.ChangePasswordRequest, *account.ChangePasswordResponse]

	profileCreate Handler[profile.CreateRequest, *profile.Response]
	profileGet    Handler[string, *profile.Response]
	profileUpdate Handler[profile.UpdateRequest, *profile.Response]
	profileDelete Handler[string, bool]
}

func (r *Router) initHandlers(s services) handlers {
	vld := validator.New()

	healthHandler := NewHandler(
		mapHealth.NewRequestMapper(),
		mapHealth.NewResponseMapper(),
		s.health.Check,
		vld,
	)

	accountRegisterHandler := NewHandler(
		mapAccount.NewRegisterRequestMapper(),
		mapAccount.NewRegisterResponseMapper(),
		s.account.Register,
		vld,
	)

	accountLoginHandler := NewHandler(
		mapAccount.NewLoginRequestMapper(),
		mapAccount.NewLoginResponseMapper(),
		s.account.Login,
		vld,
	)

	accountChangePasswordHandler := NewHandler(
		mapAccount.NewChangePasswordMapper(),
		mapAccount.NewChangePasswordResponseMapper(),
		s.account.ChangePassword,
		vld,
	)

	profileCreateHandler := NewHandler(
		mapProfile.NewCreateRequestMapper(),
		mapProfile.NewCreateResponseMapper(),
		s.profile.Create,
		vld,
	)

	profileGetHandler := NewHandler(
		mapProfile.NewGetByIdRequestMapper(),
		mapProfile.NewGetByIdResponseMapper(),
		s.profile.GetById,
		vld,
	)

	profileUpdateHandler := NewHandler(
		mapProfile.NewUpdateRequestMapper(),
		mapProfile.NewUpdateResponseMapper(),
		s.profile.Update,
		vld,
	)

	profileDeleteHandler := NewHandler(
		mapProfile.NewDeleteByIdRequestMapper(),
		mapProfile.NewDeleteByIdResponseMapper(),
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
