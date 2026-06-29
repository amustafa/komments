package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/amustafa/komments/internal/store"
)

func die(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

func getProjectRoot() string {
	if env := os.Getenv("KOMMENTS_PROJECT_ROOT"); env != "" {
		abs, err := filepath.Abs(env)
		if err == nil {
			return abs
		}
		return env
	}

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			abs, err := filepath.EvalSymlinks(root)
			if err == nil {
				return abs
			}
			return root
		}
	}

	cwd, _ := os.Getwd()
	return cwd
}

func formatPosition(pos json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(pos, &m); err != nil {
		return "?"
	}
	if m["type"] == "cursor" {
		return fmt.Sprintf("L%v", m["line"])
	}
	return fmt.Sprintf("L%v-%v", m["start_line"], m["end_line"])
}

func parsePosition(spec string) (json.RawMessage, error) {
	if parts := strings.SplitN(spec, "-", 2); len(parts) == 2 {
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid range %q — use start-end (e.g. 10-25)", spec)
		}
		return json.Marshal(map[string]any{
			"type": "range", "start_line": start, "start_col": 1,
			"end_line": end, "end_col": 1,
		})
	}

	line, err := strconv.Atoi(spec)
	if err != nil {
		return nil, fmt.Errorf("invalid position %q — use a line number (42) or range (10-25)", spec)
	}
	return json.Marshal(map[string]any{
		"type": "cursor", "line": line, "col": 1,
	})
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func usage() {
	fmt.Print(`komments — non-inline code annotations

Usage:
  komments list [--all] [--json]              List comments (active only by default)
  komments add <file> <line> <text>           Add a comment at a specific line
  komments add <file> <start>-<end> <text>    Add a comment on a line range
  komments get <id> [--json]                  Show a single comment
  komments edit <id> <text>                   Update a comment's text
  komments archive <id>                       Archive a comment
  komments unarchive <id>                     Unarchive a comment
  komments delete <id>                        Permanently delete a comment
  komments watch [--interval <seconds>]       Watch for changes and emit JSONL events

Options:
  --all              Include archived comments in list
  --json             Output as JSON
  --interval <sec>   Poll interval for watch (default: 2)
  --help             Show this help message

Examples:
  komments add src/auth.ts 42 "TODO: add rate limiting here"
  komments add src/auth.ts 10-25 "This whole block needs refactoring"
  komments list
  komments watch --interval 5
  komments archive 3
`)
	os.Exit(0)
}

func openStore() *store.Store {
	s, err := store.Open(getProjectRoot())
	if err != nil {
		die(err.Error())
	}
	return s
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func getFlagValue(args []string, flag string, defaultVal string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultVal
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 || hasFlag(args, "--help") || hasFlag(args, "-h") {
		usage()
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "list":
		cmdList(rest)
	case "add":
		cmdAdd(rest)
	case "get":
		cmdGet(rest)
	case "edit":
		cmdEdit(rest)
	case "archive":
		cmdArchive(rest)
	case "unarchive":
		cmdUnarchive(rest)
	case "delete":
		cmdDelete(rest)
	case "watch":
		cmdWatch(rest)
	default:
		die(fmt.Sprintf("unknown command %q. Run komments --help for usage.", cmd))
	}
}

func cmdList(args []string) {
	s := openStore()
	defer s.Close()

	includeArchived := hasFlag(args, "--all")
	asJSON := hasFlag(args, "--json")

	var comments []*store.Comment
	var err error
	if includeArchived {
		comments, err = s.AllComments()
	} else {
		comments, err = s.ActiveComments()
	}
	if err != nil {
		die(err.Error())
	}

	if asJSON {
		if comments == nil {
			comments = []*store.Comment{}
		}
		printJSON(comments)
		return
	}

	if len(comments) == 0 {
		if includeArchived {
			fmt.Println("No comments found.")
		} else {
			fmt.Println("No active comments. Use --all to include archived.")
		}
		return
	}

	for _, c := range comments {
		status := ""
		if c.Archived {
			status = " [archived]"
		}
		preview := strings.ReplaceAll(c.Text, "\n", " ")
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		fmt.Printf("#%d  %s:%s%s\n", c.ID, c.File, formatPosition(c.Position), status)
		fmt.Printf("    %s\n\n", preview)
	}
}

func cmdAdd(args []string) {
	if len(args) < 3 {
		die("usage: komments add <file> <line|start-end> <text>")
	}

	s := openStore()
	defer s.Close()

	filePath := args[0]
	posSpec := args[1]
	text := strings.Join(args[2:], " ")

	root := getProjectRoot()
	absPath, _ := filepath.Abs(filePath)
	relPath, err := filepath.Rel(root, absPath)
	if err != nil {
		relPath = filePath
	}

	position, err := parsePosition(posSpec)
	if err != nil {
		die(err.Error())
	}

	comment, err := s.AddComment(relPath, position, text)
	if err != nil {
		die(err.Error())
	}

	fmt.Printf("Comment #%d added at %s:%s\n", comment.ID, relPath, formatPosition(comment.Position))
}

func cmdGet(args []string) {
	if len(args) < 1 {
		die("usage: komments get <id>")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		die("usage: komments get <id>")
	}

	s := openStore()
	defer s.Close()

	comment, err := s.GetComment(id)
	if err != nil {
		die(fmt.Sprintf("comment #%d not found", id))
	}

	if hasFlag(args, "--json") {
		printJSON(comment)
		return
	}

	status := ""
	if comment.Archived {
		status = " [archived]"
	}
	fmt.Printf("#%d  %s:%s%s\n", comment.ID, comment.File, formatPosition(comment.Position), status)
	fmt.Printf("Project: %s\n", comment.ProjectRoot)
	fmt.Printf("Time:    %s\n\n", comment.Timestamp)
	fmt.Println(comment.Text)
}

func cmdEdit(args []string) {
	if len(args) < 2 {
		die("usage: komments edit <id> <new text>")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		die("usage: komments edit <id> <new text>")
	}

	s := openStore()
	defer s.Close()

	newText := strings.Join(args[1:], " ")
	ok, err := s.UpdateComment(id, newText)
	if err != nil {
		die(err.Error())
	}
	if !ok {
		die(fmt.Sprintf("comment #%d not found", id))
	}
	fmt.Printf("Comment #%d updated.\n", id)
}

func cmdArchive(args []string) {
	if len(args) < 1 {
		die("usage: komments archive <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		die("usage: komments archive <id>")
	}

	s := openStore()
	defer s.Close()

	ok, err := s.ArchiveComment(id)
	if err != nil {
		die(err.Error())
	}
	if !ok {
		die(fmt.Sprintf("comment #%d not found", id))
	}
	fmt.Printf("Comment #%d archived.\n", id)
}

func cmdUnarchive(args []string) {
	if len(args) < 1 {
		die("usage: komments unarchive <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		die("usage: komments unarchive <id>")
	}

	s := openStore()
	defer s.Close()

	ok, err := s.UnarchiveComment(id)
	if err != nil {
		die(err.Error())
	}
	if !ok {
		die(fmt.Sprintf("comment #%d not found", id))
	}
	fmt.Printf("Comment #%d unarchived.\n", id)
}

func cmdDelete(args []string) {
	if len(args) < 1 {
		die("usage: komments delete <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		die("usage: komments delete <id>")
	}

	s := openStore()
	defer s.Close()

	ok, err := s.DeleteComment(id)
	if err != nil {
		die(err.Error())
	}
	if !ok {
		die(fmt.Sprintf("comment #%d not found", id))
	}
	fmt.Printf("Comment #%d permanently deleted.\n", id)
}

type WatchEvent struct {
	Event   string         `json:"event"`
	Comment *store.Comment `json:"comment"`
}

func cmdWatch(args []string) {
	intervalStr := getFlagValue(args, "--interval", "2")
	intervalSec, err := strconv.Atoi(intervalStr)
	if err != nil || intervalSec < 1 {
		die("--interval must be a positive integer")
	}
	interval := time.Duration(intervalSec) * time.Second

	s := openStore()
	defer s.Close()

	enc := json.NewEncoder(os.Stdout)

	// Build initial snapshot
	snapshot := make(map[int]commentSnapshot)
	comments, err := s.AllComments()
	if err != nil {
		die(err.Error())
	}
	for _, c := range comments {
		snapshot[c.ID] = commentSnapshot{
			text:      c.Text,
			archived:  c.Archived,
			timestamp: c.Timestamp,
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			comments, err := s.AllComments()
			if err != nil {
				fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
				continue
			}

			currentIDs := make(map[int]bool)
			for _, c := range comments {
				currentIDs[c.ID] = true
				prev, existed := snapshot[c.ID]

				if !existed {
					enc.Encode(WatchEvent{Event: "added", Comment: c})
					snapshot[c.ID] = commentSnapshot{text: c.Text, archived: c.Archived, timestamp: c.Timestamp}
				} else if c.Archived && !prev.archived {
					enc.Encode(WatchEvent{Event: "archived", Comment: c})
					snapshot[c.ID] = commentSnapshot{text: c.Text, archived: c.Archived, timestamp: c.Timestamp}
				} else if !c.Archived && prev.archived {
					enc.Encode(WatchEvent{Event: "unarchived", Comment: c})
					snapshot[c.ID] = commentSnapshot{text: c.Text, archived: c.Archived, timestamp: c.Timestamp}
				} else if c.Text != prev.text {
					enc.Encode(WatchEvent{Event: "edited", Comment: c})
					snapshot[c.ID] = commentSnapshot{text: c.Text, archived: c.Archived, timestamp: c.Timestamp}
				}
			}

			for id, prev := range snapshot {
				if !currentIDs[id] {
					enc.Encode(WatchEvent{
						Event: "deleted",
						Comment: &store.Comment{
							ID:       id,
							Text:     prev.text,
							Archived: prev.archived,
						},
					})
					delete(snapshot, id)
				}
			}
		}
	}
}

type commentSnapshot struct {
	text      string
	archived  bool
	timestamp string
}
