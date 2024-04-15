# Production changes

## Scalability
- App is currently deployed in 1 region and all ECS tasks are running in one AZ, for world wide good performance and HA would deploy it to multiple regions and AZs
- Would have auto-scaling to handle surge in the traffic without human intervention

## Observability

- Due to time restrictions added basic metrics which allows me to get RPS and latency. In production environment there will be metric / trace interceptors that would allow
us to have more observable system
- Didn't add as it wasn't in spec, but would generate a request ID for each incoming request and refer to that in all logs

## Security
- As it wasn't in the spec I didn't implement it, but ideally this API endpoint should be accesible with a per client token, so we could know who are the clients, who are the heavy hitters
- Since it would require me to have a custom domain and a certificate, didn't implement TLS. Would run the service with TLS in production.
- In addition to rate limiter we could put a CDN for DDos protection and better performance


# API Gateway

## Code
The code for the API Gateway is in `api-gateway` directory.

## Description
Works as a transparent API Gateway between https://polygon-rpc.com. Forwards the requests and returns the response as we get it.

## Assumptions

- We're only integrated with only one network. If needed to support multiple networks
and make changes to integrate we could have another type of `Client` in `forwarder` pkg
which implements `forwarder.Client` interface
- Given I was asked to implement only 2 methods limited the RPC calls that could be done
to Polygon to the methods specified
- Given this will be an API exposed to public internet, added a basic rate limiter for all API endpoints

## Endpoints

- `/rpc`: Gets the JSONRPC request as post body, forwards request to polygon and replies with the header + response body. 
This endpoint has retry mechanism to retry 3 times for the failed requests
- `/metrics`: Endpoint for Prometheus scrapers to collect metrics for the running process
- `/health`: Health endpoint for LB usage, it is simplistic at the moment. If we were using
a database etc a ping to there could be added just as a simple check.

## How to test?

```
go test ./...
```

## How to run locally?

```
go run cmd/server/main.go
```

## How to build Docker container?

```
docker build -t trust-wallet-homework .
```

# Terraform

## Assumptions
- Due to time restriction didn't implement TLS connection to ALB as it requires a custom domain and a certificate
- Given this is just deploying one application didn't implement separate modules for VPC, ECS and LB. On a larger Terraform code base abstracting these away
with a module would be more maintainable

## How to deploy new version of the container?

### Push new version

```
aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin 471112544726.dkr.ecr.eu-west-1.amazonaws.com
docker build -t api-gateway .
docker tag api-gateway:latest 471112544726.dkr.ecr.eu-west-1.amazonaws.com/api-gateway:latest
docker push 471112544726.dkr.ecr.eu-west-1.amazonaws.com/api-gateway:latest
```

### Trigger deploy of latest version

```
aws ecs update-service --cluster api-gateway-cluster --service api-gateway-server-service --force-new-deployment
```

### Is this deployed somewhere?

Yes, to my personal AWS account. Can be tested with

```
curl --header "Content-type: application/json" --request POST --data '{
  "jsonrpc": "2.0",
  "method": "eth_getBlockByNumber",
  "params": [
    "0x134e82a",
    true
  ],
  "id": 2
}'  http://api-gateway-lb-2119393760.eu-west-1.elb.amazonaws.com/rpc
``` 