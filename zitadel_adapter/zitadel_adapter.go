/*
Copyright © 2024 Deepreo Siber Güvenlik A.S Resul ÇELİK <resul.celik@deepreo.com>
*/
package zitadel_adapter

import (
	"context"

	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/zitadel/zitadel-go/v3/pkg/client"
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
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"`
}

func (z *ZitadelAdapter) UpdateUser(ctx context.Context, userCtx *User) error {
	_, err := z.GetClient().UserServiceV2().UpdateHumanUser(ctx,
		&user.UpdateHumanUserRequest{
			UserId:   userCtx.UserID,
			Username: &userCtx.Username,
			Profile: &user.SetHumanProfile{
				GivenName:  userCtx.FirstName,
				FamilyName: userCtx.LastName,
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
