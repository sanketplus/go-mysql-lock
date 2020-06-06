# go-mysql-lock
go-mysql-lock provides locking primitive based on MySQL's [GET_LOCK](https://dev.mysql.com/doc/refman/8.0/en/locking-functions.html#function_get-lock)

### Example:

```go
package main

import (
    "context"
    "database/sql"
    
    _ "github.com/go-sql-driver/mysql"
    "github.com/sanketplus/go-mysql-lock"
)

func main() {
	db, _ := sql.Open("mysql", "root@tcp(localhost:3306)/dyno_test")

	locker := gomysqllock.NewMysqlLocker(db)

	lock, _ := locker.Obtain("foo")
	lock.Release()
}
```

#### Customizable Refresh Period
Once the lock is obtained, a goroutine periodically (default every 1 second) keeps pinging on connection since the lock is valid on a connection(session). To configure refresh interval
```go
locker := gomysqllock.NewMysqlLocker(db, WithRefreshInterval(time.Millisecond*500))
```

#### Knowing When The Lock is Lost
Obtained lock has a context which is cancelled if the lock is lost. This is determined while a goroutine keeps pinging the connection. If there is an error while pinging, assuming connection has an error, the context is cancelled. And the lock owner gets notified of the lost lock.
```go
context := lock.GetContext()
``` 