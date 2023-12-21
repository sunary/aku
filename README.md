# aku - API Gateway

`aku`' natural positioning is as a Gateway with aggregation. Its purpose is to link numerous backend services to a unified endpoint. It implements the pattern [Domain-Oriented Microservice Architecture](https://www.uber.com/en-SG/blog/microservice-architecture/), allowing you to define exactly and with a declarative configuration how is the API that you want to expose to the clients.

## features

- proxy (TCP only)
  - [x] http
  - [x] grpc
- tls
- plugin
  - [x] ip restriction
  - [ ] rate limiting
  - [ ] circuit breaker
  - [ ] cors
- tracing
  - [ ] open-telemetry
- expose metrics

## setup

- Create `akuIngress`' CRD

    ```shell
    # create a CustomResourceDefinition
    kubectl create -f artifacts/crd-aku.yaml

    # create a custom resource of type Aku
    kubectl create -f artifacts/aku.yaml
    ```

- Deploy `aku` and assign its role as a `clusterrole`.
- Configure your application's chart values behind the `aku` API gateway:

    ```yaml
    akuIngress:
      enabled: true
      routeMap:
        - name: open-prefix-public-path
          overridePath: /api/v1/your-service/public
          upstream_path: /public
        - name: open-prefix/user-path
          overridePath: /api/v1/your-service/user
          upstream_path: /user
      methodMap:
        - name: open-only-public-method
          proto_service: pb.ProtoService
          allow:
            - PublicMethod
        - name: open-all-method-except-private
          proto_service: pb.OtherProtoService
          disallow:
            - PrivateMethod
    ```

## TODO

`aku` is an experimental project, and there are many pending tasks. Here is a list of what I believe is necessary:

- [ ] define CRD.
- [ ] List of [features](#features) has been finalized.
- [ ] Once a gRPC connection is established, it remains active until a timeout occurs. This means that aku cannot control which methods are disallowed; in other words, clients can access any method when connected.
