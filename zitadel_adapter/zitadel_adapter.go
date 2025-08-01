/*
Copyright © 2024 Deepreo Siber Güvenlik A.S Resul ÇELİK <resul.celik@deepreo.com>
*/
package zitadel_adapter

import (
	"context"
	"fmt"

	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/zitadel/zitadel-go/v3/pkg/client"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/session/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

type ZitadelConfig struct {
	Domain     string `mapstructure:"domain"`
	SecretPath string `mapstructure:"secret_path"`
	PATKey     string `mapstructure:"pat_key"`
	OrgID      string `mapstructure:"org_id"`
	Port       string `mapstructure:"port"`
	Insecure   bool   `mapstructure:"insecure"`
}

type ZitadelAdapter struct {
	client *client.Client
	config *ZitadelConfig
}

var ZitadelAdapterConnect *ZitadelAdapter

func NewZitadelAdapter(ctx context.Context, config *ZitadelConfig) error {
	ztdl := new(zitadel.Zitadel)
	if config.Insecure {
		ztdl = zitadel.New(config.Domain, zitadel.WithInsecure(config.Port))
	} else {
		ztdl = zitadel.New(config.Domain)
	}
	ztdlclient := new(client.Client)
	err := error(nil)
	if config.PATKey != "" {
		ztdlclient, err = client.New(ctx, ztdl, client.WithAuth(client.PAT(config.PATKey)))
		if err != nil {
			return err
		}
	} else {
		ztdlclient, err = client.New(ctx, ztdl,
			client.WithAuth(client.DefaultServiceUserAuthentication(config.SecretPath, oidc.ScopeOpenID, client.ScopeZitadelAPI())))
		if err != nil {
			return err
		}
	}
	ZitadelAdapterConnect = &ZitadelAdapter{
		client: ztdlclient,
		config: config,
	}

	return nil
}

func (z *ZitadelAdapter) GetClient() *client.Client {
	return z.client
}

type User struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	PhoneNumber       string `json:"phone_number"`
	Email             string `json:"email"`
	PreferredLanguage string `json:"preferred_language"`
}

func (z *ZitadelAdapter) UpdateUser(ctx context.Context, userCtx *User) error {
	_, err := z.GetClient().UserServiceV2().UpdateHumanUser(ctx,
		&user.UpdateHumanUserRequest{
			UserId:   userCtx.UserID,
			Username: &userCtx.Username,
			Profile: &user.SetHumanProfile{
				GivenName:         userCtx.FirstName,
				FamilyName:        userCtx.LastName,
				PreferredLanguage: &userCtx.PreferredLanguage,
			},
			Phone: &user.SetHumanPhone{
				Phone: userCtx.PhoneNumber,
				Verification: &user.SetHumanPhone_SendCode{
					SendCode: &user.SendPhoneVerificationCode{},
				},
			},
		})
	if err != nil {
		return err
	}
	return nil
}
func (z *ZitadelAdapter) LogoutUser(ctx context.Context, userId string) error {
	resp, err := z.GetClient().SessionServiceV2().ListSessions(ctx, &session.ListSessionsRequest{
		Queries: []*session.SearchQuery{
			{
				Query: &session.SearchQuery_UserIdQuery{
					UserIdQuery: &session.UserIDQuery{
						Id: userId,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	for _, sess := range resp.Sessions {
		z.GetClient().SessionServiceV2().DeleteSession(ctx, &session.DeleteSessionRequest{
			SessionId: sess.GetId(),
		})
	}
	return nil
}

func (z *ZitadelAdapter) ListSessions(ctx context.Context) ([]*session.Session, error) { // Buraya sonradan info felan eklenebilir.
	resp, err := z.GetClient().SessionServiceV2().ListSessions(ctx, &session.ListSessionsRequest{
		Query: &object.ListQuery{},
	})
	if err != nil {
		return nil, err
	}
	fmt.Println(resp.String())
	return resp.GetSessions(), nil
}

func (z *ZitadelAdapter) UpdatePassword(ctx context.Context, userID, oldPass, newPassword string) error {
	_, err := z.GetClient().UserServiceV2().UpdateHumanUser(
		ctx,
		&user.UpdateHumanUserRequest{
			UserId: userID,
			Password: &user.SetPassword{
				PasswordType: &user.SetPassword_Password{
					Password: &user.Password{
						Password:       newPassword,
						ChangeRequired: false,
					},
				},
				Verification: &user.SetPassword_CurrentPassword{
					CurrentPassword: oldPass,
				},
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (z *ZitadelAdapter) UpdateEmail(ctx context.Context, userID, email, urlTemplate *string) error {
	_, err := z.GetClient().UserServiceV2().UpdateHumanUser(
		ctx,
		&user.UpdateHumanUserRequest{
			UserId: *userID,
			Email: &user.SetHumanEmail{
				Email: *email,
				Verification: &user.SetHumanEmail_SendCode{
					SendCode: &user.SendEmailVerificationCode{
						UrlTemplate: urlTemplate,
					},
				},
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (z *ZitadelAdapter) GetUsers(ctx context.Context) ([]User, error) {
	resp, err := z.client.UserServiceV2().ListUsers(ctx, &user.ListUsersRequest{})
	if err != nil {
		return nil, err
	}
	users := make([]User, 0)
	for _, user := range resp.Result {
		users = append(users, User{
			UserID:            user.GetUserId(),
			Username:          user.GetUsername(),
			FirstName:         user.GetHuman().GetProfile().GetGivenName(),
			LastName:          user.GetHuman().GetProfile().GetFamilyName(),
			PhoneNumber:       user.GetHuman().GetPhone().GetPhone(),
			PreferredLanguage: user.GetHuman().GetProfile().GetPreferredLanguage(),
			Email:             user.GetHuman().GetEmail().GetEmail(),
		})
	}
	return users, nil
}

func (z *ZitadelAdapter) GetUser(ctx context.Context, userID string) (*User, error) {
	resp, err := z.GetClient().UserServiceV2().GetUserByID(ctx, &user.GetUserByIDRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return &User{
		UserID:            resp.User.GetUserId(),
		Username:          resp.User.GetUsername(),
		FirstName:         resp.User.GetHuman().GetProfile().GetGivenName(),
		LastName:          resp.User.GetHuman().GetProfile().GetFamilyName(),
		PhoneNumber:       resp.User.GetHuman().GetPhone().GetPhone(),
		PreferredLanguage: resp.User.GetHuman().GetProfile().GetPreferredLanguage(),
		Email:             resp.User.GetHuman().GetEmail().GetEmail(),
	}, nil
}
