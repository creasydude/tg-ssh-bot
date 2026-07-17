package main

import (
	"fmt"
	"html"
	"strings"
)

func stripAnsi(s string) string {
	// Strip CSI sequences: ESC [ ... letter
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1B && i+1 < len(s) && s[i+1] == '[' {
			// Skip until a letter (终结符)
			j := i + 2
			for j < len(s) {
				c := s[j]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					i = j + 1
					break
				}
				j++
			}
			if j >= len(s) {
				break
			}
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func cleanOutput(raw string) string {
	cleaned := stripAnsi(raw)
	lines := strings.Split(cleaned, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func chunkMessage(text string, limit int) []string {
	if limit <= 0 {
		limit = 4000
	}
	if len(text) <= limit {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= limit {
			chunks = append(chunks, remaining)
			break
		}

		cutAt := strings.LastIndex(remaining[:limit], "\n")
		if cutAt <= 0 {
			cutAt = limit
		}

		chunks = append(chunks, remaining[:cutAt])
		remaining = strings.TrimLeft(remaining[cutAt:], "\n")
	}

	return chunks
}

func FormatOutput(cmd, output, serverInfo string, execTimeMs int64) string {
	var sb strings.Builder

	if serverInfo != "" {
		sb.WriteString(fmt.Sprintf("<b>\U0001F5A5 %s</b>\n\n", html.EscapeString(serverInfo)))
	}

	sb.WriteString(fmt.Sprintf("<code>$ %s</code>\n\n", html.EscapeString(cmd)))

	if output == "" {
		sb.WriteString("<i>(no output)</i>")
	} else {
		sb.WriteString(fmt.Sprintf("<pre>%s</pre>", html.EscapeString(output)))
	}

	if execTimeMs >= 0 {
		sb.WriteString(fmt.Sprintf("\n\n<i>%dms</i>", execTimeMs))
	}

	return sb.String()
}

func FormatError(cmd, errMsg, serverInfo string) string {
	var sb strings.Builder

	if serverInfo != "" {
		sb.WriteString(fmt.Sprintf("<b>\U0001F5A5 %s</b>\n\n", html.EscapeString(serverInfo)))
	}

	sb.WriteString(fmt.Sprintf("<code>$ %s</code>\n\n", html.EscapeString(cmd)))
	sb.WriteString(fmt.Sprintf("<i>\u274C %s</i>", html.EscapeString(errMsg)))

	return sb.String()
}
