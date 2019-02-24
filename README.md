
# Glock

Glock is a Go library that implements a distributed lock manager on top of Amazon DynamoDB.


## Installing

```
> go get github.com/danielgerlag/glock
```


## How it works

When you successfully acquire a lock, an entry is written to a table in DynamoDB that indicates the owner of the lock and the expiry time.
The default lease time is 30 seconds, each node will also send a heartbeat every 10 seconds that will renew any active leases for 30 seconds from the time of the heartbeat.
Once the expiry time has elapsed or the owning node releases the lock, it becomes available again.

## Usage

1. New up an instance of DynamoDb lock manager

	```go
	import (
		"github.com/aws/aws-sdk-go/aws"
		"github.com/danielgerlag/glock"
	)
	
	...
	config := &aws.Config{
    	Region:   aws.String("us-east-1"),
    }

    lockManager := glock.New(config, "table_name")
    ```

2. Start the heartbeat to ensure your leases are automatically renewed
    
    ```go
    lockManager.Start()
    ```
    
3. Use the `AcquireLock` and `ReleaseLock` methods to manage your distributed locks.

    ```go    
    success := lockManager.AcquireLock("my-lock-id")
    ...
    lockManager.ReleaseLock("my-lock-id")
    ```

    `AcquireLock` will return `false` if the lock is already in use and `true` if it successfully acquired the lock.
    It will start with an initial lease of 30 seconds, and a background task will renew all locally controlled leases every 10 seconds, until `ReleaseLock` is called or the application ends and the leases expire naturally.

### Notes

It is recommended that you keep the clocks of all participating nodes in sync using an NTP implementation

* https://en.wikipedia.org/wiki/Network_Time_Protocol
* https://aws.amazon.com/blogs/aws/keeping-time-with-amazon-time-sync-service/

## Authors
 * **Daniel Gerlag** - daniel@gerlag.ca

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details
