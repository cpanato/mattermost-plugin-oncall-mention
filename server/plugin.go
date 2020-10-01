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

	oncallTeamsCfg *OncallTeams
}

func (p *Plugin) getFreshOncallPeeps(mentionKey string, schedules []string, escalationManager string) ([]string, error) {
	peeps, err := p.whoIsOnCall(schedules)
	if err != nil {
		return []string{}, err
	}

	var mmUsernames []string
	if len(peeps) == 0 {
		mmUsernames = append(mmUsernames, fmt.Sprintf("@%s", escalationManager))
	} else {
		for _, peep := range peeps {
			mmUser, appErr := p.API.GetUserByEmail(peep)
			if appErr != nil {
				p.API.LogError("faild to get user", "err", appErr.Error(), "user", peep)
				continue
			}
			mmUsernames = append(mmUsernames, fmt.Sprintf("@%s", mmUser.Username))
		}
	}

	_ = p.storeOncallPersons(mentionKey, mmUsernames)

	return mmUsernames, nil
}

func (p *Plugin) storeOncallPersons(mentionKey string, oncallPersons []string) error {
	b, err := json.Marshal(oncallPersons)
	if err != nil {
		return err
	}
	appErr := p.API.KVSetWithExpiry("OnCallMention-"+mentionKey, b, 900)
	if appErr != nil {
		return fmt.Errorf("encountered error saving oncallPersons mapping")
	}
	return nil
}

func (p *Plugin) getCacheOncallPersons(mentionKey string) []string {
	oncallPersons, _ := p.API.KVGet("OnCallMention-" + mentionKey)
	var oncallPeeps []string
	err := json.Unmarshal(oncallPersons, &oncallPeeps)
	if err != nil {
		p.API.LogError("failed to decode the data from KV", "err", err.Error())
		return []string{}
	}

	return oncallPeeps
}

func (p *Plugin) MessageWillBePosted(context *plugin.Context, post *model.Post) (*model.Post, string) {
	process, err := p.Helpers.ShouldProcessMessage(post, plugin.AllowBots(), plugin.AllowSystemMessages(), plugin.AllowWebhook())
	if err != nil {
		return post, ""
	}

	if process {
		for _, oncall := range p.oncallTeamsCfg.Teams {
			if strings.Contains(post.Message, fmt.Sprintf("@%s", oncall.Mention)) {
				var oncallPeeps []string
				oncallPeeps = p.getCacheOncallPersons(oncall.Mention)
				if len(oncallPeeps) == 0 {
					p.API.LogDebug("Cache Expired or key is empty, calling opsgenie to get fresh data")
					oncallPeeps, err = p.getFreshOncallPeeps(oncall.Mention, oncall.Schedules, oncall.EscalationManager)
					if err != nil {
						p.API.LogError("failed to get fresh oncall information", "err", err.Error())
						return post, ""
					}
				}

				toReplace := fmt.Sprintf("[@%s]( \\* %s \\* )", oncall.Mention, strings.Join(oncallPeeps, " "))
				mentionsKey := fmt.Sprintf("@%s", oncall.Mention)
				newMsg := strings.Replace(post.Message, mentionsKey, toReplace, -1)
				post.Message = newMsg
			}
		}
	}

	post.Hashtags, _ = model.ParseHashtags(post.Message)
	return post, ""
}
