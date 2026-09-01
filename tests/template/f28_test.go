package template_test

import (
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/template"
)

func TestValidate_BuiltinsAllPass(t *testing.T) {
	store := newTestStore(t)
	if err := store.InstallDefaults(); err != nil {
		t.Fatal(err)
	}
	names, _ := store.List()
	for _, name := range names {
		tmpl, err := store.Get(name)
		if err != nil {
			t.Fatalf("get %s: %v", name, err)
		}
		if p := tmpl.Validate(); len(p) != 0 {
			t.Errorf("built-in %s should validate clean, got: %v", name, p)
		}
	}
}

func TestValidate_CatchesProblems(t *testing.T) {
	bad := &template.Template{
		Name:    "",
		Network: "wideopen",
		Caches:  map[string]string{"npm": "relative/path"},
		Tools:   []string{"node:"},
		Compatibility: &template.Compatibility{
			Runtimes: map[string]string{"fast": "maybe", "wat": "supported"},
		},
		Source: &template.Source{Type: "snapshot"},
	}
	problems := bad.Validate()
	joined := strings.Join(problems, " | ")
	for _, want := range []string{"name is required", "network", "absolute path", "empty name or version", "maybe", "unknown mode", "snapshotOf"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected a problem mentioning %q; got: %s", want, joined)
		}
	}
}

func TestContentDigest_StableAcrossLineageAndTimestamps(t *testing.T) {
	a := &template.Template{Name: "x", Base: "debian-slim", Network: "restricted"}
	b := &template.Template{
		Name: "x", Base: "debian-slim", Network: "restricted",
		CreatedAt: "2020-01-01T00:00:00Z",
		Lineage:   &template.Lineage{Generation: 5, ParentDigest: "sha256:abc"},
	}
	if a.ContentDigest() == "" || a.ContentDigest() != b.ContentDigest() {
		t.Fatalf("digest should ignore lineage/timestamps: %q vs %q", a.ContentDigest(), b.ContentDigest())
	}

	c := &template.Template{Name: "x", Base: "debian-slim", Network: "open"}
	if a.ContentDigest() == c.ContentDigest() {
		t.Fatal("digest should change when the definition changes")
	}
}

func TestFork_CreatesLocalDerivativeWithLineage(t *testing.T) {
	store := newTestStore(t)
	if err := store.InstallDefaults(); err != nil {
		t.Fatal(err)
	}

	forked, err := store.Fork("node", "my-node")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	if forked.Source == nil || forked.Source.Type != template.SourceLocal || forked.Source.ForkedFrom != "node" {
		t.Fatalf("source = %+v", forked.Source)
	}
	if forked.Lineage == nil || forked.Lineage.Generation != 1 || forked.Lineage.ParentDigest == "" {
		t.Fatalf("lineage = %+v", forked.Lineage)
	}

	// Persisted and re-loadable.
	reloaded, err := store.Get("my-node")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Source.ForkedFrom != "node" {
		t.Errorf("reloaded source lost: %+v", reloaded.Source)
	}
	if len(reloaded.Validate()) != 0 {
		t.Errorf("forked template should be valid: %v", reloaded.Validate())
	}

	// Fork again from the fork -> generation 2.
	gen2, err := store.Fork("my-node", "my-node-2")
	if err != nil {
		t.Fatalf("second fork: %v", err)
	}
	if gen2.Lineage.Generation != 2 {
		t.Errorf("generation = %d, want 2", gen2.Lineage.Generation)
	}

	// Name collision rejected.
	if _, err := store.Fork("node", "my-node"); err == nil {
		t.Error("forking onto an existing name should fail")
	}
}

func TestRevisions_HistoryRollbackDiff(t *testing.T) {
	store := newTestStore(t)
	if err := store.InstallDefaults(); err != nil {
		t.Fatal(err)
	}

	// Fork -> rev 1 for the new template.
	if _, err := store.Fork("go", "my-go"); err != nil {
		t.Fatal(err)
	}

	// Edit + re-write -> rev 2.
	tmpl, _ := store.Get("my-go")
	tmpl.Summary = "customized go toolchain"
	tmpl.Network = "open"
	if err := store.Create("my-go", tmpl); err != nil {
		t.Fatal(err)
	}

	hist, err := store.History("my-go")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].N != 2 {
		t.Fatalf("history = %+v, want 2 revs newest-first", hist)
	}

	// Diff rev 1 vs rev 2 shows the network change.
	r1, err := store.ResolveRef("my-go@1")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := store.ResolveRef("my-go@2")
	if err != nil {
		t.Fatal(err)
	}
	d := template.Diff(r1, r2)
	if !strings.Contains(d, "open") || !strings.Contains(d, "restricted") {
		t.Fatalf("diff missing the network change:\n%s", d)
	}

	// Rollback to rev 1 -> rev 3, network back to restricted.
	restored, err := store.Rollback("my-go", 1)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Network != "restricted" {
		t.Fatalf("rollback did not restore network: %q", restored.Network)
	}
	hist, _ = store.History("my-go")
	if len(hist) != 3 {
		t.Fatalf("rollback should add a revision; history = %+v", hist)
	}
}

func TestBundle_ExportImportRoundTrip(t *testing.T) {
	src := newTestStore(t)
	if err := src.InstallDefaults(); err != nil {
		t.Fatal(err)
	}
	orig, err := src.Fork("node", "my-node")
	if err != nil {
		t.Fatal(err)
	}

	bundle, err := template.ExportBundle(orig)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Fresh library in its own directory (newTestStore's dir name is not
	// unique, so build this one explicitly).
	dst := template.NewStore(t.TempDir())
	if err := dst.InstallDefaults(); err != nil {
		t.Fatal(err)
	}
	imported, err := dst.Import(bundle, "")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if imported.Name != "my-node" {
		t.Fatalf("name = %q", imported.Name)
	}
	if imported.Source == nil || imported.Source.Type != template.SourceImported {
		t.Fatalf("source = %+v, want imported", imported.Source)
	}
	// Definition preserved -> same content digest as the original.
	if imported.ContentDigest() != orig.ContentDigest() {
		t.Fatalf("digest drift: %s != %s", imported.ContentDigest(), orig.ContentDigest())
	}

	// Re-import onto the same name -> collision.
	if _, err := dst.Import(bundle, ""); err == nil {
		t.Fatal("re-import onto existing name should fail")
	}
	// ...but a rename works.
	if _, err := dst.Import(bundle, "my-node-copy"); err != nil {
		t.Fatalf("rename import: %v", err)
	}
}

func TestBundle_RejectsTamperedBlob(t *testing.T) {
	s := newTestStore(t)
	_ = s.InstallDefaults()
	t0, _ := s.Get("go")
	bundle, err := template.ExportBundle(t0)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a byte in the tar payload — a blob digest check must catch it.
	for i := len(bundle) / 2; i < len(bundle); i++ {
		if bundle[i] >= 'a' && bundle[i] <= 'z' {
			bundle[i] ^= 0x20
			break
		}
	}
	if _, err := template.ReadBundle(bundle); err == nil {
		t.Fatal("tampered bundle should be rejected")
	}
}

func TestBundle_RejectsNonTemplateArchive(t *testing.T) {
	if _, err := template.ReadBundle([]byte("not a tar")); err == nil {
		t.Fatal("garbage should be rejected")
	}
}
