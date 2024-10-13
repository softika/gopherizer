package api

import (
	"github.com/go-playground/validator/v10"

	"tldw/config"
	"tldw/database"

	repoAccount "tldw/database/repositories/account"
	repoProfile "tldw/database/repositories/profile"

	svcAccount "tldw/internal/services/account"
	svcHealth "tldw/internal/services/health"
	svcProfile "tldw/internal/services/profile"

	mapAccount "tldw/http/api/account"
	mapHealth "tldw/http/api/health"
	mapProfile "tldw/http/api/profile"
)

type repositories struct {
	db      database.Service
	account svcAccount.Repository
	profile svcProfile.Repository
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
	account svcAccount.Service
	health  svcHealth.Service
	profile svcProfile.Service
}

func (r *Router) initServices(cfg config.AuthConfig, s repositories) services {
	return services{
		account: svcAccount.NewService(cfg, s.account),
		health:  svcHealth.NewService(s.db),
		profile: svcProfile.NewService(s.profile),
	}
}

type handlers struct {
	health Handler[svcHealth.Request, *svcHealth.Response]

	accountRegister       Handler[svcAccount.RegisterRequest, *svcAccount.RegisterResponse]
	accountLogin          Handler[svcAccount.LoginRequest, *svcAccount.LoginResponse]
	accountChangePassword Handler[svcAccount.ChangePasswordRequest, *svcAccount.ChangePasswordResponse]

	profileCreate Handler[svcProfile.CreateRequest, *svcProfile.Response]
	profileGet    Handler[string, *svcProfile.Response]
	profileUpdate Handler[svcProfile.UpdateRequest, *svcProfile.Response]
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
