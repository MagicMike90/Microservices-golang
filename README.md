# Microservices-golang

## A micro services implemented by golang, docker

install protoc-gen-go

```go
go get -u github.com/golang/protobuf/protoc-gen-go



```

install grpc for python

```Python
pip install grpcio



pip install grpcio-tools



```

## Applied patterns

- **Proxy microservice design pattern**: This is applied using Nginx in the role of proxy. This pattern refers to the proxy for the OrchestratorNewsService, UsersService, and RecommendationService APIs.
- **Aggregator microservice design pattern**: OrchestratorNewsService performs the role of aggregator for the FamousNewsService, SportsNewsService, and PoliticsNewsServicemicroservices.
- **Branch microservice design pattern**: This is the pattern that we have used to establish communication between UsersService and RecommendationService, because RecommendationService needs information synchronously from UsersService to finish the task it proposes.
- **Asynchronous messaging microservice design pattern**: This pattern is applied between OrchestratorNewsService and RecommendationService.
