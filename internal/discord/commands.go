// internal/discord/commands.go
package discord

import (
	"strconv"
	"strings"
)

// parseCommand splits a message into command and arguments
func parseCommand(content string) (string, []string) {
	// Trim leading/trailing whitespace
	content = strings.TrimSpace(content)
	
	// Split into words
	words := strings.Fields(content)
	
	// If no words, return empty command and args
	if len(words) == 0 {
		return "", []string{}
	}
	
	// First word is the command
	command := strings.ToLower(words[0])
	
	// Rest are args
	var args []string
	if len(words) > 1 {
		args = words[1:]
	}
	
	return command, args
}

// getIntArg parses an integer argument with a default value
func getIntArg(args []string, index int, defaultVal int) int {
	if len(args) <= index {
		return defaultVal
	}
	
	val, err := strconv.Atoi(args[index])
	if err != nil {
		return defaultVal
	}
	
	return val
}

// getStringArg gets a string argument with a default value
func getStringArg(args []string, index int, defaultVal string) string {
	if len(args) <= index {
		return defaultVal
	}
	
	return args[index]
}

// checkPermission checks if a user has a permission in a channel
func checkPermission(userID, channelID, guildID string, permission int64, s *interface{}) bool {
	// TODO: Implement permission check
	// This is a placeholder that always returns true
	return true
}

// hasRole checks if a user has a specific role
func hasRole(userRoles []string, requiredRole string) bool {
	for _, role := range userRoles {
		if role == requiredRole {
			return true
		}
	}
	return false
}

// hasAnyRole checks if a user has any of the specified roles
func hasAnyRole(userRoles []string, requiredRoles []string) bool {
	for _, requiredRole := range requiredRoles {
		if hasRole(userRoles, requiredRole) {
			return true
		}
	}
	return false
}
