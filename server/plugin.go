package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate initialize the plugin
func (p *Plugin) OnActivate() error {
	p.API.LogDebug("Oncall mention plugin starting up...")

	if err := p.IsValid(p.configuration); err != nil {
		return err
	}

	p.getFreshOncallPeeps()

	p.API.LogDebug("Oncall mention plugin up.")

	return nil
}

type Oncall struct {
	Primary   string
	Secondary string
}

func (p *Plugin) getFreshOncallPeeps() Oncall {
	primary, secondary := p.whoIsOnCall("mattermost_username")

	oncallPeeps := Oncall{
		Primary:   primary,
		Secondary: secondary,
	}

	if primary == "" {
		oncallPeeps.Primary = p.configuration.ManagerEscalation
		oncallPeeps.Secondary = ""
	}

	_ = p.storeOncallPersons(oncallPeeps)

	return oncallPeeps
}

func (p *Plugin) storeOncallPersons(oncallPersons Oncall) error {
	b, err := json.Marshal(oncallPersons)
	if err != nil {
		return err
	}
	appErr := p.API.KVSetWithExpiry("OnCallMention", b, 3600)
	if appErr != nil {
		return fmt.Errorf("encountered error saving oncallPersons mapping")
	}
	return nil
}

func (p *Plugin) getCacheOncallPersons() Oncall {
	oncallPersons, _ := p.API.KVGet("OnCallMention")

	var ocallPeeps Oncall
	err := json.Unmarshal(oncallPersons, &ocallPeeps)
	if err != nil {
		return Oncall{}
	}
	return ocallPeeps
}

func (p *Plugin) IsValid(configuration *configuration) error {
	if configuration.OpsGenieAPIKey == "" {
		return fmt.Errorf("must have an OpsGenie API Key Set")
	}

	if configuration.PrimaryScheduleName == "" {
		return fmt.Errorf("must have an PrimaryScheduleName Set")
	}

	if configuration.SecondaryScheduleName == "" {
		return fmt.Errorf("must have an SecondaryScheduleName Set")
	}

	if configuration.ManagerEscalation == "" {
		return fmt.Errorf("must have an ManagerEscalation Set")
	}

	if configuration.MentionKey == "" {
		return fmt.Errorf("must have an MentionKey Set")
	}

	return nil
}

func (p *Plugin) MessageWillBePosted(context *plugin.Context, post *model.Post) (*model.Post, string) {
	process, err := p.Helpers.ShouldProcessMessage(post, plugin.AllowBots(), plugin.AllowSystemMessages(), plugin.AllowWebhook())
	if err != nil {
		return post, ""
	}

	if process {
		if strings.Contains(post.Message, fmt.Sprintf("@%s", p.configuration.MentionKey)) {
			oncallPeeps := Oncall{}
			oncallPeeps = p.getCacheOncallPersons()
			if oncallPeeps == (Oncall{}) {
				p.API.LogDebug("Cache Expired or key is empty, calling opsgenie to get fresh data")
				oncallPeeps = p.getFreshOncallPeeps()
			}
			toReplace := fmt.Sprintf("[@%s]( \\* @%s @%s \\* )", p.configuration.MentionKey, oncallPeeps.Primary, oncallPeeps.Secondary)
			if oncallPeeps.Secondary == "" {
				toReplace = fmt.Sprintf("[@%s]( \\* @%s \\* )", p.configuration.MentionKey, oncallPeeps.Primary)
			}

			mentionsKey := fmt.Sprintf("@%s", p.configuration.MentionKey)
			newMsg := strings.Replace(post.Message, mentionsKey, toReplace, -1)
			post.Message = newMsg
		}
	}

	post.Hashtags, _ = model.ParseHashtags(post.Message)

	return post, ""
}
