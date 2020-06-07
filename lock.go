package gomysqllock

import (
	"context"
	"database/sql"
	"time"
)

type Lock struct {
	key             string
	conn            *sql.Conn
	unlocker        chan (struct{})
	lostLockContext context.Context
	cancelFunc      context.CancelFunc
}

func (l Lock) GetContext() context.Context {
	return l.lostLockContext
}

func (l Lock) Release() error {
	close(l.unlocker)
	l.conn.ExecContext(context.Background(), "DO RELEASE_LOCK(?)", l.key)
	return l.conn.Close()
}

func (l Lock) refresher(duration time.Duration, cancelFunc context.CancelFunc) {
	for {
		select {
		case <-time.After(duration):
			deadline := time.Now().Add(duration)
			contextDeadline, deadlineCancelFunc := context.WithDeadline(context.Background(), deadline)
			// try refresh, else cancel
			err := l.conn.PingContext(contextDeadline)
			if err != nil {
				cancelFunc()
				deadlineCancelFunc()
				return
			}
		case <-l.unlocker:
			cancelFunc()
		}
	}
}
