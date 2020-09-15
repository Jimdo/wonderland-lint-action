# Wonderland Crons

This service manages cron jobs in Wonderland

See https://docs.jimdo-platform.net/How-To/Cron-Jobs/ for up-to-date documentation on how to use crons.

Currently, the service supports to following API calls:
 
| Verb   | URI                              | Description |
| ------ | ---------------------------------| ----------- |
| GET    | /v1/crons                        | Returns a list of all cron jobs. |
| POST   | /v1/crons                        | Runs a new cron job or updates an existing one. |
| GET    | /v1/crons/{name}                 | Returns the status of a cron job. |
| DELETE | /v1/crons/{name}                 | Stops and deletes a cron job and all of its allocations. |
| GET    | /v1/crons/{name}/allocations     | Returns a list of all cron allocations. |
| GET    | /v1/crons/allocations/{id}       | Returns the status of a cron allocation. |
| GET    | /v1/crons/allocations/{id}/logs  | Returns the log output of a cron allocation. |
