package luaprovider

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSandbox(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "helper.lua"), []byte(`return {name = "helper"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.lua"), []byte(`return {name = "secret"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.lua"), filepath.Join(dir, "link.lua")); err != nil {
		t.Fatal(err)
	}

	options := DefaultOptions()
	options.PackagePaths = []string{dir}
	p := loadTestProviderWith(t, testScript, options)

	for _, code := range []string{
		`io.popen("id")`,
		`io.open("/etc/passwd")`,
		`os.execute("id")`,
		`dofile("/etc/passwd")`,
		`loadfile("/etc/passwd")`,
		`loadstring("return 1")`,
		`load("return 1")`,
		`require("../x")`,
		`require("/etc/passwd")`,
		`require("lib/helper")`,
		`require("link")`,
		`package.path = "/etc/?"; require("passwd")`,
	} {
		if err := p.state.DoString(code); err == nil {
			t.Errorf("%s: wanted an error", code)
		}
	}

	if err := p.state.DoString(`assert(require("lib.helper").name == "helper")`); err != nil {
		t.Errorf("require inside the package path: %v", err)
	}

	p.state.SetContext(context.Background())
	if err := p.state.DoString(`print("hello", 1, nil)`); err != nil {
		t.Fatal(err)
	}
	if got := logged(p); !slices.Contains(got, "[backlog] test: hello\t1\tnil") {
		t.Errorf("print should go to the provider log, got %q", got)
	}
}
