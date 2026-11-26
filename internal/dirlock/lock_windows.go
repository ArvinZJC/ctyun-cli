/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package dirlock

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/windows"
)

// lockDirectory serializes readers and writers with a kernel mutex keyed by
// volume and directory ID. Windows releases ownership if the holding process dies.
func lockDirectory(dir *os.File, _ bool, action func() error) (err error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(dir.Fd()), &info); err != nil {
		return err
	}
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf(`Global\ctyun-directory-%08x-%08x%08x`, info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow))
	mutex, err := windows.CreateMutexEx(nil, name, 0, windows.SYNCHRONIZE|windows.MUTEX_MODIFY_STATE)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	defer func() { err = errors.Join(err, windows.CloseHandle(mutex)) }()
	// Mutex ownership belongs to the waiting OS thread, including after an abandoned wait.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if _, err := windows.WaitForSingleObject(mutex, windows.INFINITE); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, windows.ReleaseMutex(mutex)) }()
	return action()
}
