package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Pane struct {
	// TODO: include other detail for pane
	Index string
	Path  string
}

type Window struct {
	Index string
	Name  string
	Panes []Pane
}

type Session struct {
	Name    string
	Windows []Window
}

type Config struct {
	Sessions []Session
}

func runTmuxCommand(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	res, err := cmd.Output()
	return strings.TrimSpace(string(res)), err
}

func getWindowPanes(sessionName, windowName string) []Pane {
	windowPanes, err := runTmuxCommand("list-panes", "-t", sessionName+":"+windowName, "-F", "#P #{pane_current_path}")
	if err != nil {
		return nil
	}

	panes := []Pane{}
	for _, pane := range strings.Split(windowPanes, "\n") {
		paneComponents := strings.Split(pane, " ")
		p := Pane{
			Index: paneComponents[0],
			Path:  paneComponents[1],
		}

		panes = append(panes, p)
	}

	return panes
}

func getSessionWindows(sessionName string) []Window {
	sessionWindows, err := runTmuxCommand("list-windows", "-t", sessionName, "-F", "#I #W")
	if err != nil {
		return nil
	}

	windows := []Window{}
	for _, window := range strings.Split(sessionWindows, "\n") {
		windowComponents := strings.Split(window, " ")
		w := Window{
			Index: windowComponents[0],
			Name:  windowComponents[1],
			Panes: getWindowPanes(sessionName, windowComponents[1]),
		}

		windows = append(windows, w)
	}

	return windows
}

func loadCurrentState() (Config, error) {
	// sessions
	tmuxSessions, err := runTmuxCommand("ls", "-F", "#S")
	if err != nil {
		return Config{}, err
	}

	sessions := []Session{}
	for _, session := range strings.Split(tmuxSessions, "\n") {
		s := Session{
			Name:    session,
			Windows: getSessionWindows(session),
		}

		sessions = append(sessions, s)
	}

	return Config{
		Sessions: sessions,
	}, nil
}

func syncState(mainConfig *Config, currentConfig Config) error {
	return nil
}

func handleSave() error {
	f, err := os.OpenFile(".pmux.config", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .pmux.config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read from config file: %w", err)
	}

	var config Config
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("error while unmarshal: %w", err)
		}
	}

	// load current start of tmux
	config, err = loadCurrentState()
	if err != nil {
		return fmt.Errorf("failed to load current tmux state: %w", err)
	}

	// updates config
	// if err := syncState(&config, currentState); err != nil {
	// 	return errkk
	// }

	data, err = json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("faied to marshal config: %w", err)
	}

	_, err = f.Write(data)
	return err
}

func createPane(sessionName, windowName string, pane Pane) error {
	_, err := runTmuxCommand("split-window", "-t", sessionName+":"+windowName, "-c", pane.Path)
	if err != nil {
		return err
	}
	return nil
}

func createWindow(sessionName string, window Window) error {
	// TODO: add -c to specify dir path
	_, err := runTmuxCommand("new-window", "-t", sessionName, "-n", window.Name)
	if err != nil {
		return err
	}

	for _, pane := range window.Panes {
		if err := createPane(sessionName, window.Name, pane); err != nil {
			return err
		}
	}
	return nil
}

func createSession(session Session) error {
	_, err := runTmuxCommand("new", "-d", "-s", session.Name)
	if err != nil {
		return err
	}

	for _, window := range session.Windows {
		if err := createWindow(session.Name, window); err != nil {
			return err
		}

		runTmuxCommand("kill-pane", "-t", session.Name+":"+window.Name+"."+"1")
	}

	return nil
}

func replyState(config Config) error {
	for _, session := range config.Sessions {
		if err := createSession(session); err != nil {
			return err
		}

		if _, err := runTmuxCommand("kill-window", "-t", session.Name+":1"); err != nil {
			return fmt.Errorf("error deleting default window: %w", err)
		}
	}
	return nil
}

func handleRestore() error {
	f, err := os.Open(".pmux.config")
	if err != nil {
		return fmt.Errorf("failed to open .pmux.config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read from config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("faild to marshal config: %w", err)
	}

	if err := replyState(config); err != nil {
		return fmt.Errorf("failed to reply config state: %w", err)
	}

	return nil
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	if err := os.Chdir(home); err != nil {
		panic(err)
	}

	args := os.Args
	if len(args) != 2 {
		log.Fatal("incorrect number of args")
	}

	command := args[1]
	switch command {
	case "save":
		if err := handleSave(); err != nil {
			log.Fatal(err)
		}
	case "restore":
		if err := handleRestore(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("incorrect arg: %s", command)
	}
}
