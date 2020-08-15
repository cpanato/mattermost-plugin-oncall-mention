package main

import (
	"context"
	"log"
	"time"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	"github.com/opsgenie/opsgenie-go-sdk-v2/user"
)

func (p *Plugin) whoIsOnCall(userNameType string) (string, string) {

	primary, err := p.getOncall(p.configuration.PrimaryScheduleName, userNameType)
	if err != nil {
		p.API.LogError("not able to get who is the primary contact")
		return "", ""
	}

	if primary == "" {
		return "", ""
	}

	secondary, err := p.getOncall(p.configuration.SecondaryScheduleName, userNameType)
	if err != nil {
		p.API.LogError("not able to get who is the secondary contact")
		return "", ""
	}

	return primary, secondary
}

func (p *Plugin) getUserInfo(opsGenieUser, userNameType string) string {
	client, err := user.NewClient(&client.Config{
		ApiKey: p.configuration.OpsGenieAPIKey,
	})
	if err != nil {
		p.API.LogError("not able to create a new opsgenie client")
		return ""
	}

	userReq := &user.GetRequest{
		Identifier: opsGenieUser,
	}

	userResult, err := client.Get(context.Background(), userReq)
	if err != nil {
		log.Fatal("not able to get the user")
		return ""
	}

	log.Println(userResult.FullName, userResult.Details[userNameType][0])
	if len(userResult.Details[userNameType]) == 0 {
		return userResult.FullName
	}

	return userResult.Details[userNameType][0]
}

func (p *Plugin) getOncall(scheduleName, userNameType string) (string, error) {
	client, err := schedule.NewClient(&client.Config{
		ApiKey: p.configuration.OpsGenieAPIKey,
	})
	if err != nil {
		p.API.LogError("not able to create a new opsgenie client")
		return "", err
	}

	flat := true
	now := time.Now()
	onCallReq := &schedule.GetOnCallsRequest{
		Flat:                   &flat,
		Date:                   &now,
		ScheduleIdentifierType: schedule.Name,
		ScheduleIdentifier:     scheduleName,
	}
	onCall, err := client.GetOnCalls(context.TODO(), onCallReq)
	if err != nil {
		p.API.LogError("not able to GetOnCalls", "err", err.Error())
		return "", err
	}

	if (len(onCall.OnCallRecipients)) <= 0 {
		return "", nil
	}

	primary := p.getUserInfo(onCall.OnCallRecipients[0], userNameType)
	if primary == "" {
		return onCall.OnCallRecipients[0], nil
	}

	return primary, nil
}
