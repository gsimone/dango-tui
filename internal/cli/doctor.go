package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const doctorProbeTimeout = 45 * time.Second

// doctorDescribeJSON is written when cwd has no config. No describe key —
// echo pane-hook-ok is not a product default. Missing describe stays none.
const doctorDescribeJSON = "{}\n"

var errDoctorNoDescribe = fmt.Errorf("describe is none")

// RunDoctor is `dango --doctor`: stdout only, then exit. No --repo, no TUI.
func RunDoctor(w io.Writer) error {
	return Doctor(LaunchDir(), w)
}

// Doctor prints the config walk for dir, writes cwd/dango.json when cwd
// has no config file (no describe key), and probes describe when set.
func Doctor(dir string, w io.Writer) error {
	dir = filepath.Clean(dir)
	if dir == "." {
		if cwd := LaunchDir(); cwd != "" {
			dir = cwd
		}
	}
	fmt.Fprintf(w, "cwd: %s\n", dir)

	looked := configSearchPaths(dir)
	for _, path := range looked {
		fmt.Fprintf(w, "looked: %s\n", path)
	}

	if !cwdHasConfigFile(dir) {
		dest := filepath.Join(dir, "dango.json")
		if err := writeDoctorJSON(dest); err != nil {
			fmt.Fprintf(w, "won: none\n")
			fmt.Fprintf(w, "describe: none\n")
			return err
		}
		fmt.Fprintf(w, "wrote: %s\n", dest)
	}

	cfg, won, err := ReadDangoConfigAt(dir)
	if err != nil {
		fmt.Fprintf(w, "won: none\n")
		fmt.Fprintf(w, "describe: none\n")
		return err
	}
	if won == "" {
		fmt.Fprintf(w, "won: none\n")
		fmt.Fprintf(w, "describe: none\n")
		return errDoctorNoDescribe
	}
	fmt.Fprintf(w, "won: %s\n", won)

	describe := resolveDescribe(cfg.Describe, won)
	if describe == "" {
		fmt.Fprintf(w, "describe: none\n")
		return errDoctorNoDescribe
	}
	fmt.Fprintf(w, "describe: %s\n", describe)

	if out, probeErr := probeDescribe(describe, filepath.Dir(won)); probeErr != nil {
		fmt.Fprintf(w, "error: %s\n", strings.TrimSpace(probeErr.Error()))
	} else {
		fmt.Fprintf(w, "stdout: %s\n", out)
	}
	return nil
}

func writeDoctorJSON(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	_, err = io.WriteString(f, doctorDescribeJSON)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

func probeDescribe(raw, dir string) (string, error) {
	argv := strings.Fields(strings.TrimSpace(raw))
	if len(argv) == 0 {
		return "", errDoctorNoDescribe
	}
	ctx, cancel := context.WithTimeout(context.Background(), doctorProbeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes.TrimSpace(out))), nil
}
