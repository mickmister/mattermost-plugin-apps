package helloapp

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/oauther"
	"github.com/mattermost/mattermost-plugin-apps/server/api"
	"github.com/mattermost/mattermost-plugin-apps/server/constants"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

func (h *helloapp) InitOAuther() error {
	oauth2Config, err := h.GetOAuthConfig()
	if err != nil {
		return err
	}
	h.OAuther = oauther.NewFromClient(h.apps.Mattermost,
		*oauth2Config,
		h.finishOAuth2Connect,
		logger.NewNilLogger(), // TODO replace with a real logger
		oauther.OAuthURL(constants.HelloAppPath+PathOAuth2),
		oauther.StorePrefix("hello_oauth_"))
	return nil
}

func (h *helloapp) GetOAuthConfig() (*oauth2.Config, error) {
	conf := h.apps.Configurator.GetConfig()

	creds, err := h.getAppCredentials()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve App OAuth2 credentials")
	}

	return &oauth2.Config{
		ClientID:     creds.OAuth2ClientID,
		ClientSecret: creds.OAuth2ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  conf.MattermostSiteURL + "/oauth/authorize",
			TokenURL: conf.MattermostSiteURL + "/oauth/access_token",
		},
		// RedirectURL: h.AppURL(PathOAuth2Complete), - not needed, OAuther will configure
		// TODO Scopes:
	}, nil
}

func (h *helloapp) handleOAuth(w http.ResponseWriter, req *http.Request) {
	if h.OAuther == nil {
		http.Error(w, "OAuth not initialized", http.StatusInternalServerError)
		return
	}
	h.OAuther.ServeHTTP(w, req)
}

func (h *helloapp) startOAuth2Connect(userID string, callOnComplete *api.Call) (string, error) {
	state, err := json.Marshal(callOnComplete)
	if err != nil {
		return "", err
	}

	err = h.OAuther.AddPayload(userID, state)
	if err != nil {
		return "", err
	}
	return h.OAuther.GetConnectURL(), nil
}

func (h *helloapp) finishOAuth2Connect(userID string, token oauth2.Token, payload []byte) {
	c, err := api.UnmarshalCallFromData(payload)
	if err != nil {
		return
	}
	c.Context.AppID = AppID

	// TODO 2/5 we should wrap the OAuther for the users as a "service" so that
	//  - startOAuth2Connect is a Call
	//  - payload for finish should be a Call
	//  - a Call can check the presence of the acting user's OAuth2 token, and
	//    return Call startOAuth2Connect(itself)
	// for now hacking access to apps object and issuing the call from within
	// the app.

	cr, _ := h.apps.API.Call(c)

	conf := h.apps.Configurator.GetConfig()
	_ = h.apps.Mattermost.Post.DM(conf.BotUserID, c.Context.ActingUserID, &model.Post{
		Message: cr.Markdown.String(),
	})
}
