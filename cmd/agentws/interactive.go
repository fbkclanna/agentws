package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"gopkg.in/yaml.v3"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)

// --- inputModel: bubbletea model for text input with validation ---

type inputModel struct {
	textInput textinput.Model
	title     string
	validate  func(string) error
	errMsg    string
	done      bool
	aborted   bool
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			val := m.textInput.Value()
			if m.validate != nil {
				if err := m.validate(val); err != nil {
					m.errMsg = err.Error()
					return m, nil
				}
			}
			m.done = true
			return m, tea.Quit
		}
	}
	m.errMsg = ""
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title) + "\n")
	b.WriteString(m.textInput.View() + "\n")
	if m.errMsg != "" {
		b.WriteString(errStyle.Render(m.errMsg) + "\n")
	}
	return b.String()
}

// --- confirmModel: bubbletea model for yes/no confirmation ---

type confirmModel struct {
	title   string
	value   bool
	done    bool
	aborted bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "y", "Y":
			m.value = true
			m.done = true
			return m, tea.Quit
		case "n", "N":
			m.value = false
			m.done = true
			return m, tea.Quit
		case "left", "right", "tab", "h", "l":
			m.value = !m.value
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	yes := " Yes "
	no := " No "
	if m.value {
		yes = selectedStyle.Render(" Yes ")
	} else {
		no = selectedStyle.Render(" No ")
	}
	return fmt.Sprintf("%s %s / %s\n", titleStyle.Render(m.title), yes, no)
}

// --- prompt helpers ---

func promptInput(title, placeholder string, validate func(string) error) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()

	m := inputModel{
		textInput: ti,
		title:     title,
		validate:  validate,
	}

	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", err
	}
	rm := result.(inputModel)
	if rm.aborted {
		return "", fmt.Errorf("user aborted")
	}
	return rm.textInput.Value(), nil
}

func promptConfirm(title string) (bool, error) {
	m := confirmModel{
		title: title,
	}

	result, err := tea.NewProgram(m).Run()
	if err != nil {
		return false, err
	}
	rm := result.(confirmModel)
	if rm.aborted {
		return false, fmt.Errorf("user aborted")
	}
	return rm.value, nil
}

// repoIDFromURL extracts a repository name from a Git URL.
// Handles both SSH (git@host:org/repo.git) and HTTPS (https://host/org/repo.git).
func repoIDFromURL(url string) string {
	url = strings.TrimRight(url, "/")

	// SSH format: git@github.com:org/repo.git
	if idx := strings.LastIndex(url, ":"); idx != -1 && !strings.Contains(url, "://") {
		url = url[idx+1:]
	}

	// Take the last path component.
	name := path.Base(url)

	// Strip .git suffix.
	name = strings.TrimSuffix(name, ".git")

	return name
}

// interactiveAddRepos runs an interactive loop using bubbletea to collect
// repository information from the user. existingIDs prevents adding repos
// with IDs that already exist in the workspace.
func interactiveAddRepos(name, reposRoot string, existingIDs map[string]bool) ([]manifest.Repo, error) {
	var repos []manifest.Repo
	seenIDs := make(map[string]bool)
	for id := range existingIDs {
		seenIDs[id] = true
	}

	for {
		repoURL, err := promptInput(
			"Enter Git repository URL",
			"git@github.com:org/repo.git",
			func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("URL is required")
				}
				id := repoIDFromURL(s)
				if id == "" || id == "." {
					return fmt.Errorf("cannot infer repository name from URL")
				}
				if seenIDs[id] {
					return fmt.Errorf("repository ID %q is already added", id)
				}
				return nil
			},
		)
		if err != nil {
			return nil, err
		}

		repoURL = strings.TrimSpace(repoURL)
		id := repoIDFromURL(repoURL)
		repoPath := reposRoot + "/" + id

		fmt.Printf("  → id: %s, path: %s\n", id, repoPath)

		// Detect default branch with fallback.
		defaultBranch := "main"
		if b, err := git.DefaultBranch(repoURL); err == nil {
			defaultBranch = b
		} else {
			fmt.Fprintf(os.Stderr, "  warning: failed to detect default branch (%v), fallback: main\n", err)
		}

		branch, err := promptInput("Branch", defaultBranch, nil)
		if err != nil {
			return nil, err
		}

		branch = strings.TrimSpace(branch)
		if branch == "" {
			branch = defaultBranch
		}

		seenIDs[id] = true
		repos = append(repos, manifest.Repo{
			ID:      id,
			URL:     repoURL,
			Path:    repoPath,
			Ref:     branch,
			BaseRef: branch,
		})

		addMore, err := promptConfirm("Add another repository?")
		if err != nil {
			return nil, err
		}
		if !addMore {
			break
		}
	}

	return repos, nil
}

// buildWorkspace assembles a Workspace and serializes it to YAML.
func buildWorkspace(name, reposRoot, baseRef string, repos []manifest.Repo) ([]byte, error) {
	ws := manifest.Workspace{
		Version:   1,
		Name:      name,
		ReposRoot: reposRoot,
		Repos:     repos,
	}
	if baseRef != "" {
		ws.Defaults.BaseRef = baseRef
	}
	return yaml.Marshal(&ws)
}
