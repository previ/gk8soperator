# Gravitee API management Kubernetes Operator

A Kubernetes operator to manage the lifecycle of API definitions and Applications.

The operator is intended for an deployment of the API Gateway in a Kubernetes cluster, the API endpoint target can be defined as full URI resources but also as Kubernetes Services.

Only a subset of Gravitee API Gateway features are supported:

- Plans (with JWT, API Key, and Keyless security options)
- CORS
- Deployment tags

## Build and Install

Clone the repo and use the Makefile targets:

- `make client` to generate the gravitee API client stub
- `make manifests` to generate the Custom Resource Definition files
- `make install` to install the CRD in the predefined kubernetes cluster
- `make` to build the operator binary
- `make run` to run the operator locally, interacting with the predefined kubernetes cluster
- `make docker-build` to build a container image for the operator
- `helm install <release_name> --values=<your values file> helm/gk8soperator`

## CRD reference

See the [APIEndpoint](config/crd/bases/platform.my.domain_apieendpoints.yaml) and [Application](config/crd/bases/platform.my.domain_apiclients.yaml) CRD definition and examples [here](config/samples/platform_v1beta1_apiendpoint.yaml) and [here](config/samples/platform_v1beta1_apiclient.yaml) for reference.
