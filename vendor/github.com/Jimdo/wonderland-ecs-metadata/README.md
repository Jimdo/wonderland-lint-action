# Wonderland ECS Metadata Service

This repository contains Wonderland's *ECS Metadata Service* that provides metadata about AWS ECS resources. It was built because the ECS API is very strict in terms of API rate limiting. In order to perform less requests against the ECS API directly, this service is used to channel all API requests and maybe cache/aggregate queries. By channeling all API requests through this gateway, even short caching periods can make a significant impact on the number of API requests.

## API

Right now, the following API endpoints are supported:

* `GET /health` basic health check.
* `GET /cluster/<cluster name>/container-instance/<container instance arn>` returns the status of a container instance.
* `GET /cluster/<cluster name>/container-instances` returns the list of all container instances.
* `GET /cluster/<cluster name>/service/<service name>` returns the current state of a service.
* `GET /cluster/<cluster name>/task/<task arn>` returns the status of a task.
* `GET /cluster/<cluster name>/tasks/<task family>` returns the list of all tasks of a task family.
* `GET /task-definition/<task definition arn>` returns a task definition.

In the future more API endpoints can be added on demand.
