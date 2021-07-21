package keycloakoidc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
	v32 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/auth/providers/common"
	"github.com/rancher/rancher/pkg/auth/providers/oidc"
	"github.com/rancher/rancher/pkg/auth/tokens"
	client "github.com/rancher/rancher/pkg/client/generated/management/v3"
	v3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/types/config"
	"github.com/rancher/rancher/pkg/user"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Name      = "keycloakoidc"
	UserType  = "user"
	GroupType = "group"
)

type keyCloakOIDCProvider struct {
	oidc.OpenIDCProvider
}

func Configure(ctx context.Context, mgmtCtx *config.ScaledContext, userMGR user.Manager, tokenMGR *tokens.Manager) common.AuthProvider {
	return &keyCloakOIDCProvider{
		oidc.OpenIDCProvider{
			Name:        Name,
			Type:        client.KeyCloakOIDCConfigType,
			CTX:         ctx,
			AuthConfigs: mgmtCtx.Management.AuthConfigs(""),
			Secrets:     mgmtCtx.Core.Secrets(""),
			UserMGR:     userMGR,
			TokenMGR:    tokenMGR,
		},
	}
}

func (k *keyCloakOIDCProvider) GetName() string {
	return Name
}

func (k *keyCloakOIDCProvider) CustomizeSchema(schema *types.Schema) {
	schema.ActionHandler = k.ActionHandler
	schema.Formatter = k.Formatter
}

func newClient(config *v32.OIDCConfig, token *oauth2.Token) (*KeyCloakClient, error) {
	ctx, err := oidc.AddCertKeyToContext(context.Background(), config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}
	provider, err := gooidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, err
	}
	oauthConfig := oidc.ConfigToOauthConfig(provider.Endpoint(), config)
	keyCloakClient := &KeyCloakClient{
		httpClient: oauthConfig.Client(ctx, token),
	}
	return keyCloakClient, err
}

func (k *keyCloakOIDCProvider) AuthenticateUser(ctx context.Context, input interface{}) (v3.Principal, []v3.Principal, string, error) {
	login, ok := input.(*v32.OIDCLogin)
	if !ok {
		return v3.Principal{}, nil, "", fmt.Errorf("unexpected input type")
	}
	return k.LoginUser(ctx, login, nil)
}

func (k *keyCloakOIDCProvider) LoginUser(ctx context.Context, oauthLoginInfo *v32.OIDCLogin, config *v32.OIDCConfig) (v3.Principal, []v3.Principal, string, error) {
	var userPrincipal v3.Principal
	var groupPrincipals []v3.Principal
	var claimInfo oidc.ClaimInfo
	var err error

	if config == nil {
		config, err = k.GetOIDCConfig()
		if err != nil {
			return userPrincipal, groupPrincipals, "", err
		}
	}
	userInfo, oauth2Token, adminToken, err := k.GetUserInfo(&ctx, config, oauthLoginInfo.Code, &claimInfo)
	if err != nil {
		return userPrincipal, groupPrincipals, "", err
	}
	userPrincipal = k.UserToPrincipal(userInfo, claimInfo)
	userPrincipal.Me = true

	groupPrincipals, err = k.getUsersGroups(userInfo.Subject, adminToken, config, claimInfo)
	if err != nil {
		return userPrincipal, groupPrincipals, "", err
	}

	logrus.Debugf("[generic oidc] loginuser: Checking user's access to Rancher")
	allowed, err := k.UserMGR.CheckAccess(config.AccessMode, config.AllowedPrincipalIDs, userPrincipal.Name, groupPrincipals)
	if err != nil {
		return userPrincipal, groupPrincipals, "", err
	}
	if !allowed {
		return userPrincipal, groupPrincipals, "", httperror.NewAPIError(httperror.Unauthorized, "unauthorized")
	}
	// save entire oauthToken because it contains refresh_token and token expiry time
	// will use with oauth2.Client and with TokenSource to ensure auto refresh of tokens occurs for api calls
	oauthToken, err := json.Marshal(oauth2Token)
	if err != nil {
		return userPrincipal, groupPrincipals, "", err
	}
	return userPrincipal, groupPrincipals, string(oauthToken), nil
}

func (k *keyCloakOIDCProvider) RefetchGroupPrincipals(principalID string, secret string) ([]v3.Principal, error) {
	var groupPrincipals []v3.Principal
	var claimInfo oidc.ClaimInfo

	config, err := k.GetOIDCConfig()
	if err != nil {
		logrus.Errorf("[generic oidc]: error fetching OIDCConfig: %v", err)
		return groupPrincipals, err
	}
	//do not need userInfo or oauth2Token since we are only processing groups
	userInfo, _, adminToken, err := k.GetUserInfo(&k.CTX, config, secret, &claimInfo)
	if err != nil {
		return groupPrincipals, err
	}
	groupPrincipals, err = k.getUsersGroups(userInfo.Subject, adminToken, config, claimInfo)
	if err != nil {
		return groupPrincipals, err
	}
	return groupPrincipals, nil
}

func (k *keyCloakOIDCProvider) SearchPrincipals(searchValue, principalType string, token v3.Token) ([]v3.Principal, error) {
	var principals []v3.Principal
	var oauthToken *oauth2.Token
	var err error

	config, err := k.GetOIDCConfig()
	if err != nil {
		return principals, err
	}
	storedOauthToken, err := k.TokenMGR.GetSecret(token.UserID, token.AuthProvider, []*v3.Token{&token})
	if err := json.Unmarshal([]byte(storedOauthToken), &oauthToken); err != nil {
		return principals, err
	}
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		oauthToken.AccessToken = token.ProviderInfo["access_token"]
	}
	keyCloakClient, err := newClient(config, oauthToken)
	if err != nil {
		logrus.Errorf("[keycloak oidc]: error creating new http client: %v", err)
		return principals, err
	}
	accts, err := keyCloakClient.searchPrincipals(searchValue, principalType, config)
	if err != nil {
		logrus.Errorf("[keycloak oidc] problem searching keycloak: %v", err)
		return principals, err
	}
	for _, acct := range accts {
		p := k.toPrincipal(acct.Type, acct, &token)
		principals = append(principals, p)
	}
	return principals, nil
}

func (k *keyCloakOIDCProvider) toPrincipal(principalType string, acct account, token *v3.Token) v3.Principal {
	displayName := acct.Name
	if displayName == "" {
		displayName = acct.Username
	}
	princ := v3.Principal{
		ObjectMeta:  metav1.ObjectMeta{Name: k.GetName() + "_" + principalType + "://" + acct.ID},
		DisplayName: displayName,
		LoginName:   acct.Username,
		Provider:    k.GetName(),
		Me:          false,
	}

	if principalType == UserType {
		princ.PrincipalType = UserType
		if token != nil {
			princ.Me = k.IsThisUserMe(token.UserPrincipal, princ)
		}
	} else {
		princ.PrincipalType = GroupType
		princ.ObjectMeta = metav1.ObjectMeta{Name: k.GetName() + "_" + principalType + "://" + acct.Name}
		if token != nil {
			princ.MemberOf = k.TokenMGR.IsMemberOf(*token, princ)
		}
	}
	return princ
}

func (k *keyCloakOIDCProvider) GetPrincipal(principalID string, token v3.Token) (v3.Principal, error) {
	var oauthToken *oauth2.Token

	config, err := k.GetOIDCConfig()
	if err != nil {
		return v3.Principal{}, err
	}
	storedOauthToken, err := k.TokenMGR.GetSecret(token.UserID, token.AuthProvider, []*v3.Token{&token})
	if err := json.Unmarshal([]byte(storedOauthToken), &oauthToken); err != nil {
		return v3.Principal{}, err
	}
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return v3.Principal{}, err
		}
		oauthToken.AccessToken = token.ProviderInfo["access_token"]
	}
	var externalID string
	parts := strings.SplitN(principalID, ":", 2)
	if len(parts) != 2 {
		return v3.Principal{}, errors.Errorf("invalid id %v", principalID)
	}
	externalID = strings.TrimPrefix(parts[1], "//")
	parts = strings.SplitN(parts[0], "_", 2)
	if len(parts) != 2 {
		return v3.Principal{}, errors.Errorf("invalid id %v", principalID)
	}
	principalType := parts[1]
	keyCloakClient, err := newClient(config, oauthToken)
	if err != nil {
		logrus.Errorf("[keycloak oidc]: error creating new http client: %v", err)
		return v3.Principal{}, err
	}
	acct, err := keyCloakClient.getFromKeyCloakByID(externalID, principalType, config)
	if err != nil {
		return v3.Principal{}, err
	}
	princ := k.toPrincipal(principalType, acct, &token)
	return princ, err
}

// this method returns all group principals, including parent groups for a given user
func (k *keyCloakOIDCProvider) getUsersGroups(userSubjectID string, adminToken *oauth2.Token, config *v32.OIDCConfig, claimInfo oidc.ClaimInfo) ([]v3.Principal, error) {
	var groupPrincipals []v3.Principal

	if adminToken != nil {
		keyCloakClient, err := newClient(config, adminToken)
		if err != nil {
			logrus.Errorf("[keycloak oidc]: error creating new http client: %v", err)
			return groupPrincipals, err
		}
		userAndParentGroups, err := keyCloakClient.getGroupPrincipalsFromUser(userSubjectID, config)
		if err != nil {
			return groupPrincipals, err
		}
		for _, group := range userAndParentGroups {
			groupPrincipal := k.GroupToPrincipal(group.Name)
			groupPrincipal.MemberOf = true
			groupPrincipals = append(groupPrincipals, groupPrincipal)
		}
	} else {
		for _, group := range claimInfo.Groups {
			groupPrincipal := k.GroupToPrincipal(group)
			groupPrincipal.MemberOf = true
			groupPrincipals = append(groupPrincipals, groupPrincipal)
		}
	}
	return groupPrincipals, nil
}
