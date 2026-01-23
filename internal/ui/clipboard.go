package ui

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type Clipboard struct {
	message string
}

func NewClipboard() Clipboard {
	return Clipboard{}
}

func (c *Clipboard) Copy(text string) bool {
	text = ansiPattern.ReplaceAllString(text, "")
	if text == "" {
		c.message = "Nothing to copy"
		return false
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.Command("clip")
	default:
		c.message = "Copy failed"
		return false
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		c.message = "Copy failed"
		return false
	}

	c.message = "Copied!"
	return true
}

func (c *Clipboard) Message() string {
	return c.message
}

func (c *Clipboard) ClearMessage() {
	c.message = ""
}
