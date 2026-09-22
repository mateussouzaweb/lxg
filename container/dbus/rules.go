package dbus

import "strings"

// Target represents the routing destination
type Target int

const (
	TargetContainer Target = iota
	TargetHost      Target = iota
)

// Default host talk rules
var DefaultHostTalkRules = []string{
	"org.freedesktop.portal.*",
	"org.freedesktop.Notifications",
	"org.freedesktop.secrets",
	"org.freedesktop.ScreenSaver",
	"org.kde.StatusNotifierWatcher",
}

// Default host name ownership rules
var DefaultHostOwnRules = []string{
	"org.mpris.MediaPlayer2.*",
}

// MatchRule checks if a name matches a rule pattern
// Supports exact match and prefix wildcard (e.g. "org.freedesktop.portal.*")
func MatchRule(pattern string, name string) bool {
	if pattern == name {
		return true
	}

	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		if name == prefix || strings.HasPrefix(name, prefix+".") {
			return true
		}
	}

	return false
}

// RouteDestination determines whether a D-Bus destination should route to Host or Container
func RouteDestination(destination string) Target {
	if destination == "" {
		return TargetContainer
	}

	for _, rule := range DefaultHostTalkRules {
		if MatchRule(rule, destination) {
			return TargetHost
		}
	}

	return TargetContainer
}

// RouteOwnership determines whether a requested well-known name should be owned on Host or Container
func RouteOwnership(name string) Target {
	if name == "" {
		return TargetContainer
	}

	for _, rule := range DefaultHostOwnRules {
		if MatchRule(rule, name) {
			return TargetHost
		}
	}

	return TargetContainer
}

// MatchSignalRuleHost checks if a match rule likely belongs to host integration
func MatchSignalRuleHost(rule string) bool {
	for _, talk := range DefaultHostTalkRules {
		prefix := strings.TrimSuffix(talk, ".*")
		if strings.Contains(rule, prefix) {
			return true
		}
	}

	return false
}
