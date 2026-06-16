//go:build windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// readWindowsUserLocale reads the Windows user locale through the system API.
var readWindowsUserLocale = readWindowsUserLocaleFromSystem

// readWindowsUserLocaleFromSystem calls GetUserDefaultLocaleName.
func readWindowsUserLocaleFromSystem() string {
	const localeNameMaxLength = 85
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultLocaleName := kernel32.NewProc("GetUserDefaultLocaleName")
	buffer := make([]uint16, localeNameMaxLength)
	ret, _, _ := getUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if ret == 0 {
		return ""
	}
	for i, value := range buffer {
		if value == 0 {
			return string(utf16.Decode(buffer[:i]))
		}
	}
	return string(utf16.Decode(buffer))
}
