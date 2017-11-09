# Vault - Role Credential Manager (RCM)

This library manages Vault tokens and temporary AWS credentials for Go programs with Vault approles.

## Usage

In order to make use of the library, a program first needs a Vault app role ID (which can be provided by Wonderland when
an approle is configured for a service [here](../../vault_config.yaml)). With this app role ID and the address of Vault,
it is possible to run a Role Credential Manager like this:

```go
package main


import (
	"log"
	
	rcm "github.com/Jimdo/wonderland-vault/lib/role-credential-manager"
)

func main() {
    r, err := rcm.New(
        "https://vault.jimdo-platform.net",
        vaultRoleID,
    )
    if err != nil {
        log.Fatalf("Error creating RCM library: %s", err)
    }

   if err := r.Init(); err != nil {
   	log.Fatalf("Error initializing RCM: %s", err)
   }
    
    stop := make(chan interface{})
    defer close(stop)
    go func() {
        if err := r.Run(stop); err != nil {
            log.Fatalf("RCM library returned error: %s", err)
        }
    }()
    
    sec, err := r.VaultClient.Logical().Read("secret/my/services/secrets")
    // ...
}
```

In the background, RCM keeps the `VaultClient`'s Vault token up to date.

### AWS Credentials

When a service's approle is allowed to generated temporary STS credentials, RCM can keep these up to date as well:

```go
package main

import (
	"log"
	
	rcm "github.com/Jimdo/wonderland-vault/lib/role-credential-manager"
	"github.com/aws/aws-sdk-go/service/elb"
)

func main() {
    elb := elb.New(...)
    
    r, err := rcm.New(
        "https://vault.jimdo-platform.net",
        vaultRoleID,
        rcm.WithAWSIAMRole("my-services-iam-role"),
        rcm.WithAWSClientConfigs(&elb.Config),
    )
    if err != nil {
        log.Fatalf("Error initializing RCM library: %s", err)
    }
    
    stop := make(chan interface{})
    defer close(stop)
    go func() {
        if err := r.Run(stop); err != nil {
            log.Fatalf("RCM library returned error: %s", err)
        }
    }()
    
    // just use `elb` as always, credentials will always be up to date!
}
```

## Options

The first two constructor arguments (Vault Address and Vault Role ID) are mandatory, more options can be passed. To do
so, add them in the form of option functions to the constructor:

```go
package main


import (
	"log"
	
	rcm "github.com/Jimdo/wonderland-vault/lib/role-credential-manager"
	"github.com/sirupsen/logrus"
)

func main() {
    r, err := rcm.New(
        "https://vault.jimdo-platform.net",
        vaultRoleID,
        rcm.WithLogger(logrus.StandardLogger()),
    )
    if err != nil {
        log.Fatalf("Error initializing RCM library: %s", err)
    }
    
    // ...
}
```

* **`AWSIAMRole`**: The name of the service's IAM role (used for STS credentials)
* **`AWSClientConfigs`**: Config structs from the AWS SDK in which credentials should be kept up to date
* **`IgnoreErrors`**: If this option is set, errors will only be logged, but `Run` will continue to update credentials
* **`Logger`**: A logger (like for example logrus)
* **`MaxVaultRetries`**: The number of retries for Vault HTTP requests
