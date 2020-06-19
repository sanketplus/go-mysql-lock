package gomysqllock

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DefaultRefreshInterval is the periodic duration with which a connection is refreshed/pinged
const DefaultRefreshInterval = time.Second

type lockerOpt func(locker *MysqlLocker)

// MysqlLocker is the client which provide APIs to obtain lock
type MysqlLocker struct {
	db              *sql.DB
	refreshInterval time.Duration
	unlocker        chan (struct{})
}

// NewMysqlLocker returns an instance of locker which can be used to obtain locks
func NewMysqlLocker(db *sql.DB, lockerOpts ...lockerOpt) *MysqlLocker {
	locker := &MysqlLocker{
		db:              db,
		refreshInterval: DefaultRefreshInterval,
		unlocker:        make(chan (struct{})),
	}

	for _, opt := range lockerOpts {
		opt(locker)
	}

	return locker
}

// WithRefreshInterval sets the duration for refresh interval for each obtained lock
func WithRefreshInterval(d time.Duration) lockerOpt {
	return func(l *MysqlLocker) { l.refreshInterval = d }
}

// Obtain tries to acquire lock with background context. This call is expected to block is lock is already held
func (l MysqlLocker) Obtain(key string) (*Lock, error) {
	return l.ObtainContext(context.Background(), key)
}

// ObtainContext tries to acquire lock and gives up when the given context is cancelled
func (l MysqlLocker) ObtainContext(ctx context.Context, key string) (*Lock, error) {
	cancellableContext, cancelFunc := context.WithCancel(context.Background())

	dbConn, err := l.db.Conn(ctx)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("failed to get a db connection: %w", err)
	}

	row := dbConn.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1)", key)

	var res int
	errScan := row.Scan(&res)
	if errScan != nil {
		// mysql error does not tell if it was due to context closing, checking it manually
		select {
		case <-ctx.Done():
			cancelFunc()
			return nil, ErrGetLockContextCancelled
		default:
			break
		}
		cancelFunc()
		return nil, fmt.Errorf("could not read mysql response: %w", err)
	}

	lock := &Lock{
		key:             key,
		conn:            dbConn,
		unlocker:        make(chan struct{}),
		lostLockContext: cancellableContext,
		cancelFunc:      cancelFunc,
	}
	go lock.refresher(l.refreshInterval, cancelFunc)

	return lock, nil
}
