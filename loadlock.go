// +build !windows

// lockload プロセス間のロック
package lockload

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// LockHandle ロック用のハンドル
type LockHandle struct {
	fd       int
	isLocked bool
}

var (
	// ErrBusy ビジー状態エラー
	ErrBusy = errors.New("Error Lock is busy.")
	// ErrInvalidLockName ロック名が正しくない
	ErrInvalidLockName = errors.New("Invalid lock name.")
	// ErrNotInitialized ロックキーが初期化されていない
	ErrNotInitialized = errors.New("Not initialized.")
)

// InitLock ロックを初期化
func InitLock(name string) (*LockHandle, error) {
	var err error
	var fd int
	if name == "" {
		return nil, ErrInvalidLockName
	}
	tempName := filepath.Join(os.TempDir(), name)
	if _, err = os.Stat(tempName); err != nil {
		fd, err = syscall.Open(tempName, syscall.O_CREAT|syscall.O_RDONLY|syscall.O_CLOEXEC, 0644)
	} else {
		fd, err = syscall.Open(tempName, syscall.O_RDONLY|syscall.O_CLOEXEC, 0644)
	}
	if err != nil {
		return nil, err
	}
	return &LockHandle{fd: fd, isLocked: false}, nil
}

// Lock ロックを開始
func (lock *LockHandle) Lock(timeout int) error {
	if lock.fd == 0 {
		return ErrNotInitialized
	}
	now := time.Now()
	for {
		if err := syscall.Flock(lock.fd, syscall.LOCK_EX); err == nil {
			lock.isLocked = true
			return nil
		}
		time.Sleep(1 * time.Millisecond)
		if time.Since(now).Nanoseconds() > (int64(timeout) * 1000000) {
			break
		}
	}
	return ErrBusy
}

// Unlock ロック解除
func (lock *LockHandle) Unlock() error {
	if !lock.isLocked {
		return nil
	}
	if err := syscall.Flock(lock.fd, syscall.LOCK_UN); err != nil {
		return err
	}
	lock.isLocked = false
	return nil
}

// TermLock ロックを破棄
func (lock *LockHandle) TermLock() error {
	if lock.fd == 0 {
		return nil
	}
	if lock.isLocked {
		lock.Unlock()
	}
	syscall.Close(lock.fd)
	lock.fd = 0
	return nil
}
