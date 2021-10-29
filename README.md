# Relay Core

This repository contains the Kubernetes-based execution backend for
[Relay](https://relay.sh).

## Installation

In the near future, we hope to provide a Helm chart or Kustomizations to let you
easily deploy this project to your own cluster. In the meantime, please reach
out to us on [Slack](https://slack.puppet.com/) in the #relay channel and we'll
help you out!

### Requirements

* Kubernetes v1.19+
* [Tekton](https://tekton.dev/) (v0.22.0+)
* [Knative Serving](https://knative.dev/) (v0.21.0+)
* [Ambassador API Gateway](https://www.getambassador.io/docs/latest/topics/install/install-ambassador-oss/) (v1.8.0+)

## Components

### Operator

The Relay operator is responsible for reconciling the Relay [custom resource
definitions
(CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).
It is built using
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/),
although it does not use a higher-level framework like Kubebuilder or Operator
SDK.

The entry point for the operator is in
[`cmd/relay-operator`](cmd/relay-operator).

#### Resources

| API Version | Kind | Description |
|-------------|------|-------------|
| `relay.sh/v1beta1` | `Run` | Runs the defined workflow using a Tekton pipeline |
| `relay.sh/v1beta1` | `Tenant` | Defines event emission and namespace configuration for objects attached to it |
| `relay.sh/v1beta1` | `WebhookTrigger` | Creates Knative services with a given container configuration and tenant to handle webhook requests and emit events |
| `relay.sh/v1beta1` | `Workflow` | Defines a workflow using the given container configurations and dependencies |

### Metadata API

The metadata API provides runtime information to a pod running under the
supervision of the Relay operator.

The entry point for the metadata API is in
[`cmd/relay-metadata-api`](cmd/relay-metadata-api).

#### Endpoints

Requests to the metadata API are always authenticated. In production mode, we
use the source IP of the request to look up an annotation containing an
encrypted token that grants access to the resources for that pod. Once
authenticated, the following endpoints are available:

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| `GET` | `/conditions` | Any | Resolves any conditions specified in the `when` clause of a container specification |
| `POST` | `/events` | Triggers | Emits a new event using the configure trigger event sink of the pod's tenant |
| `PUT` | `/outputs/:name` | Steps | Sets the output with the given name |
| `GET` | `/outputs/:step_name/:name` | Steps | Retrieves the value of the output with the given step name and output name |
| `GET` | `/secrets/:name` | Any | Retrieves the value of the secret with the given name |
| `GET` | `/spec` | Any | Retrieves the entire specification associated with this container or a subset of the specification described by the given language (`lang`) and expression (`q`) query string parameters |
| `GET` | `/state/:name` | Any | Retrieves the value of the internal state variable with the given name |

#### Testing

To test the metadata API without deploying it in a live environment, you can run
it using a sample configuration. A selection of sample configurations are
provided in the [`examples/sample-configs`](examples/sample-configs) directory.

You can specify a JWT signing key for authenticating requests explicitly using
the `RELAY_METADATA_API_SAMPLE_HS256_SIGNING_KEY` environment variable. If not
specified, the metadata API process will generate and print one when it starts
up.

For example:

```console
$ go build -o relay-metadata-api ./cmd/relay-metadata-api
$ export RELAY_METADATA_API_SAMPLE_CONFIG_FILES=examples/sample-configs/simple.yaml
$ ./relay-metadata-api &
[...] created new HMAC-SHA256 signing key     key=[...]
[...] generated JWT for step                  run-id=1234 step-name=foo token=eyJhbGciOiJIUzI1NiJ9.[...]
[...] listening for metadata connections      addr=0.0.0.0:7000
$ curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.[...]' http://localhost:7000/spec | jq .
{
  "value": {
    "aws": {
      "accessKeyID": "AKIASAMPLEKEY",
      "secretAccessKey": "6bkpuV9fF3LX1Yo79OpfTwsw8wt5wsVLGTPJjDTu"
    },
    "foo": "bar"
  },
  "unresolvable": {},
  "complete": true
}
```

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for more information on how to
contribute to this project.
