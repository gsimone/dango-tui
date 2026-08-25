package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repoFile(t *testing.T, rel ...string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(append([]string{dir}, rel...)...)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("%s not found", filepath.Join(rel...))
		}
		dir = next
	}
}

func TestInstallShInstallsDangoAndDescribe(t *testing.T) {
	installSh := repoFile(t, "install.sh")
	fixture := repoFile(t, "testdata", "luna-describe")
	payload, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	for _, name := range []string{"dango-linux-amd64", "dango-darwin-arm64", "dango-describe"} {
		n := name
		body := []byte("#!/bin/sh\necho fake-dango\n")
		if n == "dango-describe" {
			body = payload
		}
		mux.HandleFunc("/"+n, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(body)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dest := t.TempDir()
	cmd := exec.Command("bash", installSh)
	cmd.Env = append(os.Environ(),
		"DANGO_INSTALL_DIR="+dest,
		"DANGO_NIGHTLY_BASE="+srv.URL,
		"HOME="+t.TempDir(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	got := string(out)
	dango := filepath.Join(dest, "dango")
	describe := filepath.Join(dest, "dango-describe")
	for _, path := range []string{dango, describe} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s\n%s", path, got)
		}
		if st.Mode()&0111 == 0 {
			t.Fatalf("%s must be 0755, mode %s", path, st.Mode())
		}
	}
	raw, err := os.ReadFile(describe)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(payload) {
		t.Fatalf("installed describe is not the nightly script, got %q", raw)
	}
	if !strings.Contains(got, "Installed "+dango) || !strings.Contains(got, "Installed "+describe) {
		t.Fatalf("must report both installs:\n%s", got)
	}

	t.Setenv("PATH", dest+string(os.PathListSeparator)+os.Getenv("PATH"))
	found, err := exec.LookPath("dango-describe")
	if err != nil {
		t.Fatalf("dango-describe must be on PATH after install: %v", err)
	}
	if found != describe && filepath.Clean(found) != filepath.Clean(describe) {
		t.Fatalf("LookPath dango-describe=%q want %q", found, describe)
	}
}

func TestInstallShFailsLoudWhenDescribeAssetMissing(t *testing.T) {
	installSh := repoFile(t, "install.sh")
	mux := http.NewServeMux()
	mux.HandleFunc("/dango-linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "#!/bin/sh\necho fake-dango\n")
	})
	mux.HandleFunc("/dango-darwin-arm64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "#!/bin/sh\necho fake-dango\n")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dest := t.TempDir()
	cmd := exec.Command("bash", installSh)
	cmd.Env = append(os.Environ(),
		"DANGO_INSTALL_DIR="+dest,
		"DANGO_NIGHTLY_BASE="+srv.URL,
		"HOME="+t.TempDir(),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("missing dango-describe must fail, got:\n%s", out)
	}
	got := string(out)
	if !strings.Contains(got, "no nightly binary for dango-describe") {
		t.Fatalf("must fail loud like a missing binary:\n%s", got)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "dango")); statErr == nil {
		t.Fatal("must not silently install only dango")
	}
	if _, statErr := os.Stat(filepath.Join(dest, "dango-describe")); statErr == nil {
		t.Fatal("must not install dango-describe when the asset is missing")
	}
}
