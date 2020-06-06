package gomysqllock

import "errors"

var ErrGetLockContextCancelled = errors.New("context cancelled while trying to acquire lock")
