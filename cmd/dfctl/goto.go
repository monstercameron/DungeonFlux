package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type gotoConfig struct {
	configPath string
	port       int
	dataDir    string
	pidFile    string
	server     string
}

type serverStarter func(string, []string, io.Writer, io.Writer) (*os.Process, error)

func runGoto(_ context.Context, args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) == 0 || args[0] != "goto" {
		return 0, false
	}
	if len(args) < 2 || args[1] != "combat" {
		writeError(stderr, errors.New("usage: goto combat [--config PATH] [--port N] [--data-dir PATH]"))
		return exitTransport, true
	}
	opts, err := parseGotoOptions(args[2:], stderr)
	if err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	if err := restartCombat(opts, stdout, stderr, startServer); err != nil {
		writeError(stderr, err)
		return exitTransport, true
	}
	_, _ = fmt.Fprintf(stdout, "{\"started\":true,\"phase\":\"combat.intro\",\"port\":%d}\n", opts.port)
	return exitOK, true
}

func parseGotoOptions(args []string, stderr io.Writer) (gotoConfig, error) {
	opts := gotoConfig{
		configPath: "config/fake.json",
		port:       18101,
		dataDir:    "artifacts/runtime/L-OPS",
		pidFile:    "artifacts/tmp/L-OPS/server.pid",
		server:     "go",
	}
	flags := flag.NewFlagSet("dfctl goto", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.configPath, "config", opts.configPath, "server config")
	flags.IntVar(&opts.port, "port", opts.port, "lane server port")
	flags.StringVar(&opts.dataDir, "data-dir", opts.dataDir, "runtime data directory")
	flags.StringVar(&opts.pidFile, "pid-file", opts.pidFile, "dfctl-owned server PID file")
	flags.StringVar(&opts.server, "server", opts.server, "server executable (or go)")
	if err := flags.Parse(args); err != nil {
		return gotoConfig{}, err
	}
	if flags.NArg() != 0 || opts.port < 1 || opts.port > 65535 {
		return gotoConfig{}, errors.New("invalid goto options")
	}
	return opts, nil
}

func restartCombat(opts gotoConfig, stdout, stderr io.Writer, start serverStarter) error {
	config, err := readConfig(opts.configPath)
	if err != nil {
		return err
	}
	config["debug_start"] = "combat"
	tempDir := filepath.Join("artifacts", "tmp", "L-OPS")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(opts.pidFile), 0o755); err != nil {
		return fmt.Errorf("create pid directory: %w", err)
	}
	derived := filepath.Join(tempDir, "debug-combat.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode combat config: %w", err)
	}
	if err := os.WriteFile(derived, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write combat config: %w", err)
	}
	if err := stopOwnedServer(opts.pidFile); err != nil {
		return err
	}
	args := []string{"run", "./cmd/server", "-config", derived, "-port", strconv.Itoa(opts.port), "-data-dir", opts.dataDir}
	if filepath.Ext(opts.server) != "" {
		args = []string{"-config", derived, "-port", strconv.Itoa(opts.port), "-data-dir", opts.dataDir}
	}
	process, err := start(opts.server, args, stdout, stderr)
	if err != nil {
		return fmt.Errorf("start combat server: %w", err)
	}
	if err := os.WriteFile(opts.pidFile, []byte(strconv.Itoa(process.Pid)+"\n"), 0o644); err != nil {
		_ = process.Kill()
		return fmt.Errorf("write server pid: %w", err)
	}
	return nil
}

func readConfig(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return config, nil
}

func stopOwnedServer(pidFile string) error {
	data, err := os.ReadFile(pidFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read server pid: %w", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil || pid < 1 {
		return fmt.Errorf("invalid server pid file %q", pidFile)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find server process: %w", err)
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("stop server %d: %w", pid, err)
	}
	return nil
}

func startServer(name string, args []string, stdout, stderr io.Writer) (*os.Process, error) {
	command := exec.Command(name, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	return command.Process, nil
}
