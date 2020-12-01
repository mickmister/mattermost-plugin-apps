package proxy

import (
	"github.com/pkg/errors"

	pluginapi "github.com/mattermost/mattermost-plugin-api"

	"github.com/mattermost/mattermost-plugin-apps/server/api"
	"github.com/mattermost/mattermost-server/v5/model"
)

type expander struct {
	// Context to expand (can be expanded multiple times on the same expander)
	*api.Context

	mm    *pluginapi.Client
	conf  api.Configurator
	store api.Store
}

func (p *Proxy) newExpander(cc *api.Context, mm *pluginapi.Client, conf api.Configurator, store api.Store) *expander {
	e := &expander{
		Context: cc,
		mm:      mm,
		conf:    conf,
		store:   store,
	}
	return e
}

// Expand collects the data that is requested in the expand argument, and is not
// yet collected. It then returns a new Context, filtered down to what is
// specified in expand.
func (e *expander) Expand(expand *api.Expand) (*api.Context, error) {
	clone := *e.Context
	if expand == nil {
		clone.ExpandedContext = api.ExpandedContext{}
		return &clone, nil
	}

	if expand.ActingUser != "" && e.ActingUserID != "" && e.ActingUser == nil {
		actingUser, err := e.mm.User.Get(e.ActingUserID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand acting user %s", e.ActingUserID)
		}
		e.ActingUser = actingUser
	}

	if expand.App != "" && e.AppID != "" && e.App == nil {
		app, err := e.store.LoadApp(e.AppID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand app %s", e.AppID)
		}
		e.App = app
	}

	if expand.Channel != "" && e.ChannelID != "" && e.Channel == nil {
		ch, err := e.mm.Channel.Get(e.ChannelID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand channel %s", e.ChannelID)
		}
		e.Channel = ch
	}

	// Config is cached pre-sanitized
	if expand.Config != "" && e.Config == nil {
		mmconf := e.conf.GetMattermostConfig()
		e.Config = &api.MattermostConfig{}
		if mmconf.ServiceSettings.SiteURL != nil {
			e.Config.SiteURL = *mmconf.ServiceSettings.SiteURL
		}
	}

	// TODO expand Mentioned

	if expand.Post != "" && e.PostID != "" && e.Post == nil {
		post, err := e.mm.Post.GetPost(e.PostID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand post %s", e.PostID)
		}
		e.Post = post
	}

	if expand.RootPost != "" && e.RootPostID != "" && e.RootPost == nil {
		post, err := e.mm.Post.GetPost(e.RootPostID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand root post %s", e.RootPostID)
		}
		e.RootPost = post
	}

	if expand.Team != "" && e.TeamID != "" && e.Team == nil {
		team, err := e.mm.Team.Get(e.TeamID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand team %s", e.TeamID)
		}
		e.Team = team
	}

	if expand.User != "" && e.UserID != "" && e.User == nil {
		user, err := e.mm.User.Get(e.UserID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to expand user %s", e.UserID)
		}
		e.User = user
	}

	// Insert the access tokens
	// <><>TODO insert user OAuth token if applicable

	clone.ExpandedContext = api.ExpandedContext{
		ActingUser: e.stripUser(e.ActingUser, expand.ActingUser),
		App:        e.stripApp(expand.App),
		Channel:    e.stripChannel(expand.Channel),
		Config:     e.stripConfig(expand.Config),
		Post:       e.stripPost(e.Post, expand.Post),
		RootPost:   e.stripPost(e.RootPost, expand.RootPost),
		Team:       e.stripTeam(expand.Team),
		User:       e.stripUser(e.User, expand.User),
		// TODO Mentioned
	}
	return &clone, nil
}

func (e *expander) stripUser(user *model.User, level api.ExpandLevel) *model.User {
	if user == nil || level == api.ExpandAll {
		return user
	}
	if level != api.ExpandSummary {
		return nil
	}
	return &model.User{
		BotDescription: user.BotDescription,
		DeleteAt:       user.DeleteAt,
		Email:          user.Email,
		FirstName:      user.FirstName,
		Id:             user.Id,
		IsBot:          user.IsBot,
		LastName:       user.LastName,
		Locale:         user.Locale,
		Nickname:       user.Nickname,
		Roles:          user.Roles,
		Timezone:       user.Timezone,
		Username:       user.Username,
	}
}

func (e *expander) stripChannel(level api.ExpandLevel) *model.Channel {
	if e.Channel == nil || level == api.ExpandAll {
		return e.Channel
	}
	if level != api.ExpandSummary {
		return nil
	}
	return &model.Channel{
		Id:          e.Channel.Id,
		DeleteAt:    e.Channel.DeleteAt,
		TeamId:      e.Channel.TeamId,
		Type:        e.Channel.Type,
		DisplayName: e.Channel.DisplayName,
		Name:        e.Channel.Name,
	}
}

func (e *expander) stripTeam(level api.ExpandLevel) *model.Team {
	if e.Team == nil || level == api.ExpandAll {
		return e.Team
	}
	if level != api.ExpandSummary {
		return nil
	}
	return &model.Team{
		Id:          e.Team.Id,
		DisplayName: e.Team.DisplayName,
		Name:        e.Team.Name,
		Description: e.Team.Description,
		Email:       e.Team.Email,
		Type:        e.Team.Type,
	}
}

func (e *expander) stripPost(post *model.Post, level api.ExpandLevel) *model.Post {
	if post == nil || level == api.ExpandAll {
		return post
	}
	if level != api.ExpandSummary {
		return nil
	}
	return &model.Post{
		Id:        e.Post.Id,
		Type:      e.Post.Type,
		UserId:    e.Post.UserId,
		ChannelId: e.Post.ChannelId,
		RootId:    e.Post.RootId,
		Message:   e.Post.Message,
	}
}

func (e *expander) stripApp(level api.ExpandLevel) *api.App {
	if e.App == nil {
		return nil
	}

	app := *e.App
	app.Secret = ""
	app.OAuth2ClientSecret = ""

	switch level {
	case api.ExpandAll, api.ExpandSummary:
		return &app
	}
	return nil
}

func (e *expander) stripConfig(level api.ExpandLevel) *api.MattermostConfig {
	if e.Config == nil {
		return nil
	}
	switch level {
	case api.ExpandAll, api.ExpandSummary:
		return e.Config
	}
	return nil
}