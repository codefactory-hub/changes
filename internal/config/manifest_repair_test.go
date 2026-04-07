package config

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestRepoLayoutManifestWriterMatchesInitSymbolicForms(t *testing.T) {
	repoRoot := t.TempDir()

	for _, tc := range []struct {
		name      string
		selection RepoInitSelection
		wantRoot  string
		wantCfg   string
		wantData  string
		wantState string
	}{
		{
			name: "xdg",
			selection: RepoInitSelection{
				Style:  StyleXDG,
				Root:   repoRoot,
				Config: filepath.Join(repoRoot, ".config", "changes"),
				Data:   filepath.Join(repoRoot, ".local", "share", "changes"),
				State:  filepath.Join(repoRoot, ".local", "state", "changes"),
			},
			wantRoot:  "$REPO_ROOT",
			wantCfg:   "$REPO_ROOT/.config/changes",
			wantData:  "$REPO_ROOT/.local/share/changes",
			wantState: "$REPO_ROOT/.local/state/changes",
		},
		{
			name: "home",
			selection: RepoInitSelection{
				Style:  StyleHome,
				Root:   filepath.Join(repoRoot, ".changes"),
				Config: filepath.Join(repoRoot, ".changes", "config"),
				Data:   filepath.Join(repoRoot, ".changes", "data"),
				State:  filepath.Join(repoRoot, ".changes", "state"),
			},
			wantRoot:  "$REPO_ROOT/.changes",
			wantCfg:   "$layout.root/config",
			wantData:  "$layout.root/data",
			wantState: "$layout.root/state",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := WriteRepoLayoutManifest(tc.selection, repoRoot)
			if err != nil {
				t.Fatalf("WriteRepoLayoutManifest returned error: %v", err)
			}

			var doc layoutDocument
			if _, err := toml.Decode(string(raw), &doc); err != nil {
				t.Fatalf("decode manifest: %v", err)
			}

			if doc.Layout.Root != tc.wantRoot {
				t.Fatalf("root = %q, want %q", doc.Layout.Root, tc.wantRoot)
			}
			if doc.Layout.Config != tc.wantCfg {
				t.Fatalf("config = %q, want %q", doc.Layout.Config, tc.wantCfg)
			}
			if doc.Layout.Data != tc.wantData {
				t.Fatalf("data = %q, want %q", doc.Layout.Data, tc.wantData)
			}
			if doc.Layout.State != tc.wantState {
				t.Fatalf("state = %q, want %q", doc.Layout.State, tc.wantState)
			}
		})
	}
}
