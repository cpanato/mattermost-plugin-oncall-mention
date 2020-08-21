# Plugin OnCall Mention

This Mattermost Plugin helps to have a single mention to call who is onCall in the time you make a post using the configured mention.

Today it works with OpsGenie and in a very strict OpsGenie Schedule configuration that we use inside Mattermost and inside a specific Team.
However there is a plan to make this generic, so any feedback and PRs are welcome.

## Configuration

- Install this Plugin in your Mattermost instance
- Get the OpsGenie API Key https://docs.opsgenie.com/docs/api-key-management
- Go to `System Console > Plugins > OnCall Mention` in your Mattermost instance
    - Set the OpsGenie API Key
    - Set the Teams that are on call for that you need to use the following [oncall.json](oncall.json) as example

    ```json
    {
        "teams": [
            {
                "team": "team-B",
                "mention": "peeps-oncall",
                "schedules": [
                    "my-opsgenie-team-schedule",
                    "my-second-opsgenie-team-schedule"
                ],
                "escalation_manager": "MM_manager_username"
            },
            {
                "team": "support-team",
                "mention": "support-oncall",
                "schedules": [
                    "my-opsgenie-team-support-schedule"
                ],
                "escalation_manager": "MM_manager_username"
            }
        ]
    }
    ```

Where:

- `team`: The team name for the configuration
- `mention`: is the string that when you posting a message wil be replaced to the persons that are on call
- `schedules`: are the schedules you have configures in OpsGenie for a particular Team.
- `escalation_manager`: When there is no one on call in the moment the mention was triggered or any error ocurred it will use the manager for that mention/team. Needs to be the Mattermost username for the manager.

## Usage

if you set the `@mention` to be something like `oncall-peeps` everytime you metion this it will convert to a link and add the users that are oncall at that moment.

For example, lets say that `user-1` and `user-2` are onCall when you create a post with `@oncall-peeps looks like my application on cluster ABC is not working` it will convert the `@oncall-peeps` to a link and add the users to that, so the users will receive an notification.

If you edit the message you will see the plugin made the change and will be something like `[@oncall-peeps]( \\* @user-1 @user-2 \\*) looks like my application on cluster ABC is not working`

## Development

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.

Use `make check-style` to check the style.

Use `make deploy` to deploy the plugin to your local server. Before running `make deploy` you need to set a few environment variables:

```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
```