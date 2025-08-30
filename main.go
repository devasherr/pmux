package main

import (
	"encoding/json"
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
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var config Config
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	}

	// load current start of tmux
	config, err = loadCurrentState()
	if err != nil {
		return err
	}

	// updates config
	// if err := syncState(&config, currentState); err != nil {
	// 	return errkk
	// }

	data, err = json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}

func handleRestore() {}

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
			// log.Fatal("failed to save tmux state: ", err)
		}
	case "restore":
		handleRestore()
	default:
		log.Fatalf("incorrect arg: %s", command)
	}
}
