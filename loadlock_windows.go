// +build windows

// lockload プロセス間のロック
package lockload

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// LockHandle ミューテックスハンドルを保持する。
type LockHandle struct {
	mutex    uintptr
	isLocked bool
}

// DLLハンドル
type dllHandle struct {
	dll *syscall.DLL
}

var (
	kernel32Dll         = loadDLL("kernel32.dll")
	createMutexW        = kernel32Dll.findProc("CreateMutexW")
	waitForSingleObject = kernel32Dll.findProc("WaitForSingleObject")
	releaseMutex        = kernel32Dll.findProc("ReleaseMutex")
	closeHandle         = kernel32Dll.findProc("CloseHandle")
)

const (
	waitObject0   int = 0
	waitAbandoned int = 128
	waitTimeout   int = 258
)

func loadDLL(name string) *dllHandle {
	dll, err := syscall.LoadDLL(name)
	if err != nil {
		panic(err)
	}
	return &dllHandle{dll: dll}
}

func (handle *dllHandle) findProc(name string) *syscall.Proc {
	proc, err := handle.dll.FindProc(name)
	if err != nil {
		panic(err)
	}
	return proc
}

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
	mutexName := fmt.Sprintf("Global\\%s", name)
	mutex, _, _ := createMutexW.Call(
		0, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(mutexName))))
	if mutex == 0 {
		err := syscall.GetLastError()
		return nil, err
	}
	return &LockHandle{mutex: mutex, isLocked: false}, nil
}

// Lock ロックを開始
func (lock *LockHandle) Lock(timeout int) error {
	handle, _, _ := waitForSingleObject.Call(lock.mutex, uintptr(timeout))
	if int(handle) == waitObject0 || int(handle) == waitAbandoned {
		// Lock成功
		lock.isLocked = true
		return nil
	} else if int(handle) == waitTimeout {
		return ErrBusy
	}
	return fmt.Errorf("Unknown Error. [%v]", syscall.GetLastError())
}

// Unlock ロックを解除
func (lock *LockHandle) Unlock() error {
	if !lock.isLocked {
		return nil
	}
	handle, _, _ := releaseMutex.Call(lock.mutex)
	if int(handle) == 0 { // 失敗
		return fmt.Errorf("Unlock Error. [%v]", syscall.GetLastError())
	}
	lock.isLocked = false
	return nil
}

// IsLocked 自プロセスがロックしているかの確認
func (lock *LockHandle) IsLocked() bool {
	isLocked := false
	handle, _, _ := waitForSingleObject.Call(lock.mutex, 0)
	if int(handle) == waitObject0 || int(handle) == waitAbandoned {
		isLocked = lock.isLocked
		releaseMutex.Call(lock.mutex)
		return isLocked
	}
	return isLocked
}

// TermLock ミューテックスを破棄
func (lock *LockHandle) TermLock() error {
	closeHandle.Call(lock.mutex)
	return nil
}
