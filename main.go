package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/git"
	"github.com/ihatemodels/gdev/internal/store"
	"github.com/ihatemodels/gdev/internal/ui/app"
	"github.com/ihatemodels/gdev/internal/ui/styles"
)

var Version = "dev"

func main() {
	startView := parseArgs()
	if startView < 0 {
		return
	}

	s, err := store.New()
	if err != nil {
		fmt.Println(styles.Error.Render("Error: failed to initialize store"))
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(s)
	if err != nil {
		fmt.Println(styles.Error.Render("Error: failed to load config"))
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	ri := loadRepoInfo(s)

	if startView == app.TodosView && ri == nil {
		fmt.Println(styles.Error.Render("Error: not in a git repository"))
		fmt.Println("TODO management requires a git repository.")
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(s, cfg, ri, Version, startView), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func parseArgs() app.View {
	if len(os.Args) <= 1 {
		return app.MainMenuView
	}

	switch os.Args[1] {
	case "todo", "todos":
		return app.TodosView
	case "help", "--help", "-h":
		printHelp()
		return -1
	case "version", "--version", "-v":
		fmt.Printf("gdev %s\n", Version)
		return -1
	default:
		return app.MainMenuView
	}
}

func printHelp() {
	fmt.Println("Usage: gdev [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  todo    Start directly in TODO management")
	fmt.Println("  help    Show this help message")
	fmt.Println()
	fmt.Println("Run without arguments to show the main menu.")
}

func loadRepoInfo(s *store.Store) *app.RepoInfo {
	repo, err := git.GetRepo()
	if err != nil {
		return nil
	}

	ri := &app.RepoInfo{Repo: repo}

	state, err := s.TouchRepo(repo.Root, repo.Name)
	if err == nil {
		ri.State = state
	}

	ri.Ahead, ri.Behind, _ = repo.GetAheadBehind()
	ri.HasChanges, _ = repo.HasLocalChanges()

	return ri
}
