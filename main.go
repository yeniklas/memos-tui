package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yeniklas/memos-tui/internal/api"
	"github.com/yeniklas/memos-tui/internal/config"
	"github.com/yeniklas/memos-tui/internal/model"
	"github.com/yeniklas/memos-tui/internal/tui"
)

var version = "dev"

func main() {
	profile := flag.String("profile", "", "config profile to use (default: default_profile from config)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	journalFlag := flag.Bool("journal", false, "open or create today's journal entry")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	prof, err := config.Load(*profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *journalFlag {
		client := api.New(prof.URL, prof.Token)
		if err := runJournal(client, prof); err != nil {
			fmt.Fprintln(os.Stderr, "journal:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	theme, err := config.LoadTheme()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tui.ApplyTheme(theme)

	client := api.New(prof.URL, prof.Token)
	app := tui.NewApp(client, prof.MarkdownEnabled(), theme)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runJournal(client *api.Client, prof config.Profile) error {
	date := time.Now().Format("2006-01-02")
	tag := prof.JournalTagOrDefault()
	title := "# " + date

	filter := fmt.Sprintf(`content.contains("%s") && content.contains("#%s")`, title, tag)
	memos, _, err := client.ListMemos(filter, "", false)
	if err != nil {
		return err
	}

	var existing *model.Memo
	for i := range memos {
		for _, line := range strings.Split(memos[i].Content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if line == title {
				existing = &memos[i]
			}
			break
		}
		if existing != nil {
			break
		}
	}

	initial := title + "\n\n#" + tag + "\n"
	if existing != nil {
		initial = existing.Content
	}

	tmp, err := os.CreateTemp("", "memos-tui-journal-*.md")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(initial); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()

	tmpPath := tmp.Name()
	initialHash := sha256.Sum256([]byte(initial))

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	content, err := os.ReadFile(tmpPath)
	os.Remove(tmpPath)
	if err != nil {
		return err
	}

	newHash := sha256.Sum256(content)
	if newHash == initialHash || strings.TrimSpace(string(content)) == "" {
		return nil
	}

	if existing == nil {
		_, err = client.CreateMemo(string(content), model.VisibilityPrivate)
	} else {
		_, err = client.UpdateMemo(existing.Name, string(content))
	}
	return err
}
