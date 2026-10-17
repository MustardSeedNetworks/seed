package deviceconfig

// IOS and IOS-XE offer no dry run, so Seed predicts the change from the fresh
// backup (ADR-0035): the candidate's lines folded into the running
// configuration's sections the way the parser would take them. The prediction
// is labelled as such; the post-check after apply is what proves the result.
// A replace makes the candidate the whole configuration, so its prediction is
// the candidate itself.

import (
	"strings"
)

// iosNode is one configuration line and the lines indented beneath it.
type iosNode struct {
	line     string
	children []*iosNode
}

// predictIOS predicts the diff a candidate would make to backup's running
// configuration. A target without an archive path is refused: the revert
// timer apply depends on needs one. So is a replace that would remove it,
// which would leave the next push refused.
func predictIOS(backup Backup, mode PushMode, candidate string) (DryRunResult, error) {
	const refusal = "the revert timer needs an archive path; configure \"archive\" with \" path flash:seed-archive\""
	running := parseIOS(backup.Config)
	if !iosHasArchivePath(running) {
		return DryRunResult{}, &PlanRefusedError{Reason: refusal + " first"}
	}
	if mode == PushMerge {
		return lineDiff("running", renderIOS(running), "predicted", renderIOS(mergeIOS(running, parseIOS(candidate))))
	}
	next := parseIOS(iosConfigOnly(candidate))
	if !iosHasArchivePath(next) {
		return DryRunResult{}, &PlanRefusedError{Reason: refusal + " in the replacement too"}
	}
	return lineDiff("running", renderIOS(parseIOS(iosConfigOnly(backup.Config))), "candidate", renderIOS(next))
}

// iosConfigOnly drops what "show running-config" prints around and between
// the configuration: its banner lines, comments, and the final "end". A
// replacement is compared with the running configuration without them, so a
// candidate saved from "show running-config" an hour ago is not a change.
func iosConfigOnly(text string) string {
	var b strings.Builder
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "!"), trimmed == "end", trimmed == "Building configuration...",
			strings.HasPrefix(trimmed, "Current configuration :"):
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// parseIOS reads a configuration into its sections by indentation. Comment
// separators ("!") and blank lines carry nothing and are dropped from both
// sides of a prediction alike.
func parseIOS(text string) []*iosNode {
	type frame struct {
		indent int
		node   *iosNode
	}
	var (
		roots []*iosNode
		stack []frame
	)
	for raw := range strings.SplitSeq(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || trimmed == "!" {
			continue
		}
		indent := len(line) - len(trimmed)
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		n := &iosNode{line: trimmed}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			parent := stack[len(stack)-1].node
			parent.children = append(parent.children, n)
		}
		stack = append(stack, frame{indent: indent, node: n})
	}
	return roots
}

func renderIOS(nodes []*iosNode) string {
	var b strings.Builder
	var walk func([]*iosNode, int)
	walk = func(ns []*iosNode, depth int) {
		for _, n := range ns {
			b.WriteString(strings.Repeat(" ", depth))
			b.WriteString(n.line)
			b.WriteByte('\n')
			walk(n.children, depth+1)
		}
	}
	walk(nodes, 0)
	return b.String()
}

func iosHasArchivePath(nodes []*iosNode) bool {
	for _, n := range nodes {
		if n.line != "archive" {
			continue
		}
		for _, c := range n.children {
			if strings.HasPrefix(c.line, "path ") {
				return true
			}
		}
	}
	return false
}

// mergeIOS folds the candidate's lines into running, returning a new tree and
// leaving running as it was. Each candidate line, within its section:
//
//   - "no X" removes X, or a line that is X with arguments;
//   - a line already present merges its children into the existing one;
//   - X takes the place of "no X", and a singleton of its predecessor;
//   - anything else is appended, at the top level before the final "end".
func mergeIOS(running, candidate []*iosNode) []*iosNode {
	out := cloneIOS(running)
	for _, c := range candidate {
		out = mergeIOSLine(out, c)
	}
	return out
}

func mergeIOSLine(siblings []*iosNode, c *iosNode) []*iosNode {
	if c.line == "end" {
		return siblings
	}
	if negated, ok := strings.CutPrefix(c.line, "no "); ok {
		return removeIOS(siblings, func(n *iosNode) bool {
			return n.line == negated || strings.HasPrefix(n.line, negated+" ")
		})
	}
	for _, n := range siblings {
		if n.line == c.line {
			for _, cc := range c.children {
				n.children = mergeIOSLine(n.children, cc)
			}
			return siblings
		}
	}
	added := cloneIOS([]*iosNode{c})[0]
	for i, n := range siblings {
		if n.line == "no "+c.line || sameSingleton(n.line, c.line) {
			siblings[i] = added
			return siblings
		}
	}
	if last := len(siblings) - 1; last >= 0 && siblings[last].line == "end" {
		return append(siblings[:last], added, siblings[last])
	}
	return append(siblings, added)
}

// sameSingleton reports whether a and b set the same command of which a
// section holds one, so entering b replaces a rather than adding a second.
func sameSingleton(a, b string) bool {
	for _, prefix := range []string{
		"hostname ",
		"description ",
		"ip domain-name ",
		"ip domain name ",
		"switchport mode ",
		"switchport access vlan ",
		"switchport trunk native vlan ",
		"speed ",
		"duplex ",
		"mtu ",
		"enable secret ",
	} {
		if strings.HasPrefix(a, prefix) && strings.HasPrefix(b, prefix) {
			return true
		}
	}
	return false
}

func removeIOS(nodes []*iosNode, drop func(*iosNode) bool) []*iosNode {
	out := make([]*iosNode, 0, len(nodes))
	for _, n := range nodes {
		if !drop(n) {
			out = append(out, n)
		}
	}
	return out
}

func cloneIOS(nodes []*iosNode) []*iosNode {
	out := make([]*iosNode, len(nodes))
	for i, n := range nodes {
		out[i] = &iosNode{line: n.line, children: cloneIOS(n.children)}
	}
	return out
}
