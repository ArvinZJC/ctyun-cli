//go:build !windows

package cli

var readWindowsUserLocale = func() string {
	return ""
}
