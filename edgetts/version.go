package edgetts

// Version information.
const Version = "7.2.8"

var versionInfo = [3]int{7, 2, 8}

// VersionInfo returns the semantic version as [major, minor, patch].
func VersionInfo() [3]int { return versionInfo }
