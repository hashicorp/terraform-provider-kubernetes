## File Storage Directories SDK for API version 2018-11-09

This package allows you to interact with the Directories File Storage API

### Supported Authorizers

* Azure Active Directory (for the Resource Endpoint `https://storage.azure.com`)
* SharedKeyLite (Blob, File & Queue)

### Example Usage

```go
package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/Azure/go-autorest/autorest"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/file/directories"
)

func Example() error {
	accountName := "storageaccount1"
    storageAccountKey := "ABC123...."
    shareName := "myshare"
    directoryName := "myfiles"
    
    storageAuth := autorest.NewSharedKeyLiteAuthorizer(accountName, storageAccountKey)
    directoriesClient := directories.New()
    directoriesClient.Client.Authorizer = storageAuth
    
    ctx := context.TODO()
    metadata := map[string]string{
    	"hello": "world",
    }
    if _, err := directoriesClient.Create(ctx, accountName, shareName, directoryName, metadata); err != nil {
        return fmt.Errorf("Error creating Directory: %s", err)
    }
    
    return nil 
}
```