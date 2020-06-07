package gomysqllock

import "errors"

// ErrGetLockContextCancelled is returned when user given context is cancelled while trying to obtain the lock
var ErrGetLockContextCancelled = errors.New("context cancelled while trying to obtain lock")
