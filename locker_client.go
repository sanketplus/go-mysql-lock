package gomysqllock

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const DefaultRefreshInterval = time.Second

type LockerOpt func(locker *MysqlLocker)

type MysqlLocker struct {
	db              *sql.DB
	refreshInterval time.Duration
	unlocker        chan (struct{})
}

type MysqlLockerInterface interface {
	Obtain(string) (context.Context, error)
	ObtainContext(context.Context, string) (context.Context, error)
}

func NewMysqlLocker(db *sql.DB, lockerOpts ...LockerOpt) *MysqlLocker {
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

func WithRefreshInterval(d time.Duration) LockerOpt {
	return func(l *MysqlLocker) { l.refreshInterval = d }
}

func (l MysqlLocker) Obtain(key string) (*Lock, error) {
	return l.ObtainContext(context.Background(), key)
}

func (l MysqlLocker) ObtainContext(ctx context.Context, key string) (*Lock, error) {
	cancellableContext, cancelFunc := context.WithCancel(context.Background())

	dbConn, err := l.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get a db connection: %w", err)
	}

	row := dbConn.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1)", key)

	var res int
	errScan := row.Scan(&res)
	if errScan != nil {
		// mysql error does not tell if it was due to context closing, checking it manually
		select {
		case <-ctx.Done():
			return nil, ErrGetLockContextCancelled
		default:
			break
		}
		return nil, fmt.Errorf("could not read mysql response: %w", err)
	}

	lock := &Lock{
		key:             key,
		conn:            dbConn,
		unlocker:        make(chan (struct{})),
		lostLockContext: cancellableContext,
		cancelFunc:      cancelFunc,
	}
	go lock.refresher(l.refreshInterval, cancelFunc)

	return lock, nil
}
