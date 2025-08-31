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
	PaneIndex string
}

type Window struct {
	// TODO: include other details like index because window name own its own is not unique
	WindowName  string
	WindowPanes []Pane
}

type Session struct {
	SessionName    string
	SessionWindows []Window
}

type Config struct {
	Sessions []Session
}

func runTmuxCommand(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	res, err := cmd.Output()
	return strings.TrimSpace(string(res)), err
}

func getWindowPanes(windowName string) []Pane {
	windowPanes, err := runTmuxCommand("list-panes", "-t", strings.TrimRight(windowName, "*-"))
	if err != nil {
		return nil
	}

	panes := []Pane{}
	for _, pane := range strings.Split(windowPanes, "\n") {
		p := Pane{
			PaneIndex: strings.Split(pane, ":")[0],
		}

		panes = append(panes, p)
	}

	return panes
}

func getSessionWindows(sessionName string) []Window {
	sessionWindows, err := runTmuxCommand("list-windows", "-t", sessionName)
	if err != nil {
		return nil
	}

	windows := []Window{}
	for _, window := range strings.Split(sessionWindows, "\n") {
		windowName := strings.Split(window, ":")[1]
		windowName = strings.TrimSpace(windowName)

		idx := 0
		for windowName[idx] != ' ' {
			idx++
		}

		// TODO: also include the path the window is at
		w := Window{
			WindowName:  strings.TrimRight(windowName[:idx], "*-"),
			WindowPanes: getWindowPanes(sessionName + ":" + windowName[:idx]),
		}

		windows = append(windows, w)
	}

	return windows
}

func loadCurrentState() (Config, error) {
	// sessions
	tmuxSessions, err := runTmuxCommand("ls")
	if err != nil {
		return Config{}, err
	}

	sessions := []Session{}
	for _, session := range strings.Split(tmuxSessions, "\n") {
		sessionName := strings.Split(session, ":")[0]
		s := Session{
			SessionName:    sessionName,
			SessionWindows: getSessionWindows(sessionName),
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
	f, err := os.OpenFile(".pmux.config", os.O_RDWR|os.O_CREATE, 0644)
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

func createPane(sessionName, windowName string, _ Pane) error {
	_, err := runTmuxCommand("split-window", "-t", sessionName+":"+windowName)
	if err != nil {
		return err
	}
	return nil
}

func createWindow(sessionName string, window Window) error {
	// TODO: add -c to specify dir path
	_, err := runTmuxCommand("new-window", "-t", sessionName, "-n", window.WindowName)
	if err != nil {
		return err
	}

	for _, pane := range window.WindowPanes {
		if err := createPane(sessionName, window.WindowName, pane); err != nil {
			return err
		}
	}
	return nil
}

func createSession(session Session) error {
	_, err := runTmuxCommand("new", "-d", "-s", session.SessionName)
	if err != nil {
		return err
	}

	for _, window := range session.SessionWindows {
		if err := createWindow(session.SessionName, window); err != nil {
			return err
		}
	}

	return nil
}

func replyState(config Config) error {
	for _, session := range config.Sessions {
		if err := createSession(session); err != nil {
			return err
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
