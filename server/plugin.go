package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

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
	CacheTime time.Time
}

func (p *Plugin) getFreshOncallPeeps() Oncall {
	primary, secondary := p.whoIsOnCall("mattermost_username")

	oncallPeeps := Oncall{
		Primary:   primary,
		Secondary: secondary,
		CacheTime: time.Now(),
	}

	if primary == "" {
		oncallPeeps.Primary = p.configuration.ManagerEscalation
		oncallPeeps.Secondary = ""
	}

	p.storeOncallPersons(oncallPeeps)

	return oncallPeeps
}

func (p *Plugin) storeOncallPersons(oncallPersons Oncall) error {
	b, err := json.Marshal(oncallPersons)
	if err != nil {
		return err
	}
	err = p.API.KVSet("OnCallMention", []byte(b))
	if err != nil {
		return fmt.Errorf("Encountered error saving oncallPersons mapping")
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
		return fmt.Errorf("Must have an OpsGenie API Key Set")
	}

	if configuration.PrimaryScheduleName == "" {
		return fmt.Errorf("Must have an PrimaryScheduleName Set")
	}

	if configuration.SecondaryScheduleName == "" {
		return fmt.Errorf("Must have an SecondaryScheduleName Set")
	}

	if configuration.ManagerEscalation == "" {
		return fmt.Errorf("Must have an ManagerEscalation Set")
	}

	if configuration.MentionKey == "" {
		return fmt.Errorf("Must have an MentionKey Set")
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
				oncallPeeps = p.getFreshOncallPeeps()
			}
			oneHourCache := oncallPeeps.CacheTime.Add(1 * time.Hour)
			testCacheTime := time.Now().Sub(oneHourCache)
			if testCacheTime >= 0 {
				p.API.LogDebug("Cache Expired, calling opsgenie to get fresh data")
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
