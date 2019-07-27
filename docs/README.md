= Tasks

This repository builds several task containers which can be used in the
`steps` section of the Nebula workflow YAML.

== Slack Notification

Example Usage:

```yaml
steps:
  - name: awesome-step
    ...
  - name: notify
    dependsOn: [awesome-step]
    image: gcr.io/nebula-235818/nebula-slack-notification:latest
    spec:
      apitoken: !Secret slack-token
      channel: "#channel-name"
      message: "Message to print"
```

Limitations: currently the notification will always run even if
`dependsOn` step(s) fail. Ideally we could customize the message with
information about any other steps in the workflow. However, we need a
way to obtain that information.
