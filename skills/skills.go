// Package skills holds what tells an agent how to use the command line.
package skills

import _ "embed"

// MeetKeep is the skill that reads meetings with the meetkeep command.
//
//go:embed meetkeep/SKILL.md
var MeetKeep string
