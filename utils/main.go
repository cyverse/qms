package utils

import "regexp"

// usernameSuffixRegexp is a regular expression that can be used to remove suffixes from usernames.
var usernameSuffixRegexp = regexp.MustCompile("@.*$")

// RemoveUsernameSuffix removes the suffix from a username. The Discovery Environment is the only CyVerse product that
// uses a username suffix, so this suffix will be removed in order to ensure that the same subscription and usage
// information is shared across all CyVerse products.
func RemoveUsernameSuffix(username string) string {
	return usernameSuffixRegexp.ReplaceAllString(username, "")
}
