# Aku - API Gateway

## support

- [x] http
- [x] grpc

## create CRD

```shell
# create a CustomResourceDefinition
kubectl create -f artifacts/crd-aku.yaml

# create a custom resource of type Aku
kubectl create -f artifacts/aku.yaml
```

set aku as `clusterrole`
