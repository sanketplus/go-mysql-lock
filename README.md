# go-mysql-lock
[![GoDoc](https://godoc.org/github.com/sanketplus/go-mysql-lock?status.svg)](https://godoc.org/github.com/sanketplus/go-mysql-lock)
[![Azure DevOps builds](https://img.shields.io/azure-devops/build/sanketplus/go-mysql-lock/1)](https://dev.azure.com/sanketplus/go-mysql-lock/_build?definitionId=1)
[![Azure DevOps coverage](https://img.shields.io/azure-devops/coverage/sanketplus/go-mysql-lock/1)](https://dev.azure.com/sanketplus/go-mysql-lock/_build?definitionId=1)
[![Go Report Card](https://goreportcard.com/badge/github.com/sanketplus/go-mysql-lock)](https://goreportcard.com/report/github.com/sanketplus/go-mysql-lock)

go-mysql-lock provides locking primitive based on MySQL's [GET_LOCK](https://dev.mysql.com/doc/refman/8.0/en/locking-functions.html#function_get-lock)
Lock names are strings and MySQL enforces a maximum length on lock names of 64 characters.

## Use cases
Though there are mature locking primitives provided by systems like Zookeeper and etcd, when you have an application which
is primarily dependent on MySQL for its uptime and health, added resiliency provided by systems just mentioned doesn't add
much benefit. go-mysql-lock helps when you have multiple application instances which are backed by a common mysql instance
and you want only one of those application instances to hold a lock and do certain tasks.

#### Installation
```$bash
go get github.com/sanketplus/go-mysql-lock
```

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

## Features
#### Customizable Refresh Period
Once the lock is obtained, a goroutine periodically (default every 1 second) keeps pinging on connection since the lock
is valid on a connection(session). To configure the refresh interval
```go
locker := gomysqllock.NewMysqlLocker(db, gomysqllock.WithRefreshInterval(time.Millisecond*500))
```

#### Obtain Lock With Context
By default, an attempt to obtain a lock is backed by background context. That means the `Obtain` call would block
indefinitely. Optionally, an `Obtain` call can be made with user given context which will get cancelled with the given
context. The following call will give up after a second if the lock was not obtained.
```go
locker := gomysqllock.NewMysqlLocker(db)
ctxShort, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
lock, err := locker.ObtainContext(ctxShort, "key")
```
#### Know When The Lock is Lost
Obtained lock has a context which is cancelled if the lock is lost. This is determined while a goroutine keeps pinging the connection. If there is an error while pinging, assuming connection has an error, the context is cancelled. And the lock owner gets notified of the lost lock.
```go
context := lock.GetContext()
``` 