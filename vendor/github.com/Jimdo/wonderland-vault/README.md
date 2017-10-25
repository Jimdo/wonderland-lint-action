[![jenkins stage](https://badges.fyi/static/Jenkins/stage)](https://jenkins.jimdo-platform-stage.net/job/Vault-Deploy/)
[![papertrail stage](https://badges.fyi/static/papertrail/stage/03498f)](https://papertrailapp.com/groups/1772563/events?q=system%3Avault)
[![jenkins prod](https://badges.fyi/static/Jenkins/prod)](https://jenkins.jimdo-platform.net/job/Vault-Deploy/)
[![papertrail prod](https://badges.fyi/static/papertrail/prod/03498f)](https://papertrailapp.com/groups/1401924/events?q=system%3Avault)

# Jimdo / wonderland-vault

This repository contains the configuration for [Vault](https://www.vaultproject.io/) in the Wonderland.

## Parts

- AWS Roles for later use of the [AWS secret backend](https://www.vaultproject.io/docs/secrets/aws/index.html)
- Vault ACLs to allow users to access our Vault based on [Github teams](https://github.com/orgs/Jimdo/teams)
- Basic configuration of our Vault instance to have everything documented in code
- A wrapper library which makes it more easy to use AppRole inside Go applications

## Enabling access to our Vault for your team

If your team doesn't have access to our Vault yet you need to create a pull request in this repository changing `vault_acls.yaml`:

Lets take the entry for the "Developers" Team as an example:

```yaml
keys:
  - key: sys/policy/developers
    values:
      rules: |
        path "secret/graylog/*" { capabilities = ["create", "read", "update", "delete", "list"] }
        path "secret/developers/*" { capabilities = ["create", "read", "update", "delete", "list"] }

  - key: auth/github/map/teams/Developers
    values: {value: developers}
```

These entries
- creates a policy called `developers` an a mapping for the GitHub team `Developers` to the policy `developers`.
- allows users with that policy to read and write to all secrets below `secret/graylog/` and `secret/developers/`

Though it's possible to use `policy = "write"` instead of `capabilities` we prefer to use the newer and more fine grained `capabilities` in our configuration. To gain an overview which capabilities are available for you to use please see the [Vault documentation](https://www.vaultproject.io/docs/concepts/policies.html) about ACLs.

## Wonderland internals
### How to create initial database dump

This section documents how the initial dump was created. This should never be required because it rotates the unseal keys and the root token. Normally it should be sufficient to delete the table and restore the initial dump if everything got fucked up.

1. Stop Vault completely
1. Delete `wonderland-vault` table
1. Start Vault again, do **NOT** unseal it
1. Do a `vault init -key-shares=5 -key-threshold=2`, note down unseal keys / root token
1. Run `make backup_initial`

### Create AppRole access for applications including AWS access

You need to create three things:

- The AWS role in `vault_aws.yaml`
- The ACL containing access to the AWS and secret paths your app requires in `vault_acls.yaml`:  
```hcl
path "secret/wonderland/infrastructure/crims" { capabilities = ["read", "list"] }
path "aws/sts/wonderland-service-pricing" { capabilities = ["read", "list"] }
```
- The AppRole in `vault_config.yaml`

For a more specific example see the configuration for the `wonderland-service-pricing` service. It contains all three parts mentioned here.
