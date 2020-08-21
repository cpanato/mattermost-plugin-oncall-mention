package main

import (
	"context"
	"time"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
	"github.com/pkg/errors"
)

func (p *Plugin) whoIsOnCall(schedules []string) ([]string, error) {
	client, err := schedule.NewClient(&client.Config{
		ApiKey: p.configuration.OpsGenieAPIKey,
	})
	if err != nil {
		p.API.LogError("not able to create a new opsgenie client")
		return []string{}, err
	}
	var oncallPeeps []string
	for _, schedule := range schedules {
		oncallPeepsSchedule, err := p.getOncall(client, schedule)
		if err != nil {
			p.API.LogError("not able to get who is oncall")
			return oncallPeeps, errors.Wrap(err, "failed to get who is oncall")
		}
		oncallPeeps = appendUniqueValues(oncallPeeps, oncallPeepsSchedule)
	}

	return oncallPeeps, nil
}

func appendUniqueValues(a []string, b []string) []string {

	check := make(map[string]int)
	d := append(a, b...)
	res := make([]string, 0)
	for _, val := range d {
		check[val] = 1
	}

	for value := range check {
		res = append(res, value)
	}

	return res
}

func (p *Plugin) getOncall(client *schedule.Client, scheduleName string) ([]string, error) {
	flat := true
	now := time.Now().UTC()
	onCallReq := &schedule.GetOnCallsRequest{
		Flat:                   &flat,
		Date:                   &now,
		ScheduleIdentifierType: schedule.Name,
		ScheduleIdentifier:     scheduleName,
	}
	onCall, err := client.GetOnCalls(context.TODO(), onCallReq)
	if err != nil {
		p.API.LogError("not able to GetOnCalls", "err", err.Error())
		return []string{}, err
	}

	if (len(onCall.OnCallRecipients)) <= 0 {
		return []string{}, nil
	}

	var users []string
	for _, onCallRecipient := range onCall.OnCallRecipients {
		users = append(users, onCallRecipient)
	}

	return users, nil
}
