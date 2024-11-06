/*
Copyright © 2024 Deepreo Siber Güvenlik A.S Resul ÇELİK <resul.celik@deepreo.com>
*/
package zitadel_auth

import (
	"context"
	"net/http"

	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"
	"github.com/zitadel/zitadel-go/v3/pkg/http/middleware"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var Zit *Zitadel

type ZitadelConfig struct {
	Domain     string `mapstructure:"domain"`
	SecretPath string `mapstructure:"secret_path"`
	Port       string `mapstructure:"port"`
	Insecure   bool   `mapstructure:"insecure"`
}

type Zitadel struct {
	authenticator *authorization.Authorizer[*oauth.IntrospectionContext]
	middleware    *middleware.Interceptor[*oauth.IntrospectionContext]
}

func NewZitadel(ctx context.Context, config ZitadelConfig) (zit *Zitadel, err error) {
	ztdl := new(zitadel.Zitadel)
	if config.Insecure {
		ztdl = zitadel.New(config.Domain, zitadel.WithInsecure(config.Port))
	} else {
		ztdl = zitadel.New(config.Domain)
	}
	authn, err := authorization.New(ctx, ztdl, oauth.DefaultAuthorization(config.SecretPath))
	if err != nil {
		return nil, err
	}
	mid := middleware.New(authn)
	Zit = &Zitadel{
		authenticator: authn,
		middleware:    mid,
	}

	return Zit, nil
}

func (z *Zitadel) GetAuthenticatorRoute() *authorization.Authorizer[*oauth.IntrospectionContext] {
	return z.authenticator
}

func (z *Zitadel) GetMiddleware() *middleware.Interceptor[*oauth.IntrospectionContext] {
	return z.middleware
}

func (z *Zitadel) AuthenticatorMiddleware(next http.Handler) http.Handler {
	return z.middleware.RequireAuthorization()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		}),
	)
}

func (z *Zitadel) GetUserInfo(ctx context.Context) *oauth.IntrospectionContext {
	return z.middleware.Context(ctx)
}

// return func(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
// 		ctx, err := i.authorizer.CheckAuthorization(req.Context(), req.Header.Get(authorization.HeaderName), options...)
// 		if err != nil {
// 			if errors.Is(err, &authorization.UnauthorizedErr{}) {
// 				http.Error(w, err.Error(), http.StatusUnauthorized)
// 				return
// 			}
// 			http.Error(w, err.Error(), http.StatusForbidden)
// 			return
// 		}
// 		req = req.WithContext(authorization.WithAuthContext(req.Context(), ctx))
// 		next.ServeHTTP(w, req)
// 	})
// }
