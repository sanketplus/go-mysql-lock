package gomysqllock

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func setupDB_oldDB(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", "root@tcp(localhost:3305)/")
	assert.NoError(t, err, "failed to setup db")
	return db
}

func getLockContext_oldDB(ctx context.Context, t *testing.T, key string, db *sql.DB) *Lock {
	locker := NewMysqlLocker(db)
	l, err := locker.ObtainTimeoutContext(ctx, key, 100000)
	assert.NoError(t, err, "failed to obtain lock")
	return l
}

func getLock_oldDB(t *testing.T, key string, db *sql.DB) *Lock {
	locker := NewMysqlLocker(db)
	l, err := locker.ObtainTimeout(key, 10)
	assert.NoError(t, err, "failed to obtain lock")
	return l
}

func releaseLock_oldDB(t *testing.T, l *Lock) {
	err := l.Release()
	assert.NoError(t, err, "failed to release lock")
}

func TestMysqlLocker_LockContext_oldDB_Success(t *testing.T) {
	ctx := context.Background()
	db := setupDB_oldDB(t)
	key := "foo"
	lock := getLockContext_oldDB(ctx, t, key, db)
	lockContext := lock.GetContext()
	releaseLock_oldDB(t, lock)

	// making sure lock's context is done after lock is released
	select {
	case <-lockContext.Done():
	default:
		assert.Fail(t, "lock's context is not cancelled after lock is released")
	}
}

func TestMysqlLocker_LockContext_oldDB_Timeout(t *testing.T) {
	db := setupDB_oldDB(t)
	locker := NewMysqlLocker(db, WithRefreshInterval(time.Millisecond*500))
	key := "bar"

	// obtain lock
	lock := getLock_oldDB(t, key, db)

	// try to get the same lock with timeout context
	ctxShort, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	_, err := locker.ObtainTimeoutContext(ctxShort, key, 10)

	cancelFunc()
	assert.Equal(t, ErrGetLockContextCancelled, err)

	releaseLock_oldDB(t, lock)
}

func TestMysqlLocker_DBError_oldDB_AfterLock(t *testing.T) {
	db := setupDB_oldDB(t)
	key := "baz"

	// obtain lock
	lock := getLock_oldDB(t, key, db)
	lockContext := lock.GetContext()

	// perhaps also simulate db crash
	lock.conn.Close()

	// sleeping so that periodic refresher (running 1 sec by default) cancels the context
	time.Sleep(time.Second * 2)

	// making sure lock's context is done after db is closed
	select {
	case <-lockContext.Done():
		assert.Contains(t, lockContext.Err().Error(), "context canceled")
	default:
		assert.Fail(t, "lock's context is not cancelled after lock is released")
	}
}

func TestMysqlLocker_Obtain_oldDB_DBError(t *testing.T) {
	// broken db connection
	db, _ := sql.Open("mysql", "root@tcp(localhost:33006)/")
	locker := NewMysqlLocker(db)

	_, err := locker.Obtain("test")
	assert.Contains(t, err.Error(), "failed to get a db connection")
}

func TestMysqlLocker_Obtain_oldDB_DBScanError(t *testing.T) {
	db, _ := sql.Open("mysql", "root@tcp(localhost:3305)/")
	locker := NewMysqlLocker(db)

	// setting very long key name shall result into error
	_, err := locker.Obtain(strings.Repeat("x", 100))
	assert.Contains(t, err.Error(), "internal mysql error acquiring the lock")
}
