// +build windows

// lockload プロセス間のロック
package lockload

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

// LockHandle ミューテックスハンドルを保持する。
type LockHandle struct {
	handle      windows.Handle
	mutexHandle windows.Handle
	name        *uint16
	mu          sync.Mutex
}

const (
	waitObject0   int = 0
	waitAbandoned int = 128
	waitTimeout   int = 258
)

var (
	// ErrBusy ビジー状態エラー
	ErrBusy = errors.New("Error Lock is busy.")
	// ErrInvalidLockName ロック名が正しくない
	ErrInvalidLockName = errors.New("Invalid lock name.")
	// ErrNotInitialized ロックキーが初期化されていない
	ErrNotInitialized = errors.New("Not initialized.")
)

// InitLock ロックを作成
func InitLock(name string) (*LockHandle, error) {
	if name == "" {
		return nil, ErrInvalidLockName
	}
	mutexName := syscall.StringToUTF16Ptr(fmt.Sprintf("Global\\%s", name))
	//handle, err := windows.CreateMutex(nil, false, mutexName)
	/*if err != nil {
		handle = 0
	}*/
	return &LockHandle{name: mutexName, handle: 0, mutexHandle: 0}, nil
}

// Lock ロックを開始
func (lock *LockHandle) Lock() error {
	runtime.LockOSThread()
	lock.mu.Lock()
	handle, err := windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, lock.name)
	if err != nil {
		if err.Error() != "The system cannot find the file specified." {
			fmt.Println(err)
			return err
		}
		lock.mutexHandle, err = windows.CreateMutex(nil, false, lock.name)
		if err != nil {
			fmt.Println(err)
			return err
		}
		handle, err = windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, lock.name)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	event, err := windows.WaitForSingleObject(handle, 500)
	if int(event) == waitObject0 || int(event) == waitAbandoned {
		lock.handle = handle
		return nil
	}
	fmt.Println(err)
	return ErrBusy
}

// Unlock ロックを解除
func (lock *LockHandle) Unlock() error {
	defer func() {
		lock.mu.Unlock()
		runtime.UnlockOSThread()
	}()
	err1 := windows.ReleaseMutex(lock.handle)
	err2 := windows.CloseHandle(lock.handle)
	if err1 != nil {
		fmt.Println(err1)
		return err1
	}
	if err2 != nil {
		fmt.Println(err2)
		return err2
	}
	return nil
}

func (lock *LockHandle) Term() {
	if lock.mutexHandle != 0 {
		windows.CloseHandle(lock.mutexHandle)
	}
}
