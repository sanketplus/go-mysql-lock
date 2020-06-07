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

func setupDB(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", "root@tcp(localhost:3306)/")
	assert.NoError(t, err, "failed to setup db")
	return db
}

func getLockContext(t *testing.T, key string, db *sql.DB, ctx context.Context) *Lock {
	locker := NewMysqlLocker(db)
	l, err := locker.ObtainContext(ctx, key)
	assert.NoError(t, err, "failed to obtain lock")
	return l
}

func getLock(t *testing.T, key string, db *sql.DB) *Lock {
	locker := NewMysqlLocker(db)
	l, err := locker.Obtain(key)
	assert.NoError(t, err, "failed to obtain lock")
	return l
}

func releaseLock(t *testing.T, l *Lock) {
	err := l.Release()
	assert.NoError(t, err, "failed to release lock")
}

func TestMysqlLocker_LockContext_Success(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)
	key := "foo"
	lock := getLockContext(t, key, db, ctx)
	lockContext := lock.GetContext()
	releaseLock(t, lock)

	// making sure lock's context is done after lock is released
	select {
	case <-lockContext.Done():
	default:
		assert.Fail(t, "lock's context is not cancelled after lock is released")
	}
}

func TestMysqlLocker_LockContext_Timeout(t *testing.T) {
	db := setupDB(t)
	locker := NewMysqlLocker(db, WithRefreshInterval(time.Millisecond*500))
	key := "bar"

	// obtain lock
	lock := getLock(t, key, db)

	// try to get the same lock with timeout context
	ctxShort, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	_, err := locker.ObtainContext(ctxShort, key)

	cancelFunc()
	assert.Equal(t, ErrGetLockContextCancelled, err)

	releaseLock(t, lock)
}

func TestMysqlLocker_DBError_AfterLock(t *testing.T) {
	db := setupDB(t)
	key := "baz"

	// obtain lock
	lock := getLock(t, key, db)
	lockContext := lock.GetContext()

	// perhaps also simulate db crash
	lock.conn.Close()

	// sleeping so that periodic refresher (running 1 sec by default) cancels the context
	time.Sleep(time.Second * 2)

	// making sure lock's context is done after db is closed
	select {
	case <-lockContext.Done():
	default:
		assert.Fail(t, "lock's context is not cancelled after lock is released")
	}
}

func TestMysqlLocker_Obtain_DBError(t *testing.T) {
	// broken db connection
	db, _ := sql.Open("mysql", "root@tcp(localhost:33006)/")
	locker := NewMysqlLocker(db)

	_, err := locker.Obtain("test")
	assert.Contains(t, err.Error(), "failed to get a db connection")
}

func TestMysqlLocker_Obtain_DBScanError(t *testing.T) {
	db, _ := sql.Open("mysql", "root@tcp(localhost:3306)/")
	locker := NewMysqlLocker(db)

	// setting very long key name shall result into error
	_, err := locker.Obtain(strings.Repeat("x", 100))
	assert.Contains(t, err.Error(), "could not read mysql response")
}
