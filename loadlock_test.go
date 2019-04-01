// +build !windows

package lockload

import (
	"testing"
)

func TestInitLock(t *testing.T) {
	if _, err := InitLock(""); err == nil {
		t.Error("lock name should be invalid. ")
	} else if err != ErrInvalidLockName {
		t.Errorf("lock name should be invalid. [%s]", err.Error())
	}

}
