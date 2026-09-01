package template

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Bundle media types (ADR-008). vnd.pi-sandbox.template.* so a bundle is a
// valid OCI artifact but distinguishable from a container image.
const (
	mediaManifest   = "application/vnd.oci.image.manifest.v1+json"
	mediaIndex      = "application/vnd.oci.image.index.v1+json"
	artifactType    = "application/vnd.pi-sandbox.template.manifest.v1"
	mediaConfig     = "application/vnd.pi-sandbox.template.config.v1+json"
	mediaDefinition = "application/vnd.pi-sandbox.template.definition.v1+yaml"
)

// BundleExporterVersion identifies the tool that wrote a bundle.
var BundleExporterVersion = "pi-box/dev"

// BundleProvenance is the config blob of a template bundle.
type BundleProvenance struct {
	ContentDigest string   `json:"contentDigest"`
	Source        *Source  `json:"source,omitempty"`
	Lineage       *Lineage `json:"lineage,omitempty"`
	Exporter      string   `json:"exporter"`
	ExportedAt    string   `json:"exportedAt"`
}

type ociDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ociManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	ArtifactType  string            `json:"artifactType,omitempty"`
	Config        ociDescriptor     `json:"config"`
	Layers        []ociDescriptor   `json:"layers"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

type ociIndex struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Manifests     []ociDescriptor `json:"manifests"`
}

func blobDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ExportBundle writes an OCI image layout (ADR-008) for a template and
// returns the tar bytes. Artifact layers are out of scope for this
// version; a bundle is definition + provenance only.
func ExportBundle(t *Template) ([]byte, error) {
	if problems := t.Validate(); len(problems) > 0 {
		return nil, fmt.Errorf("template does not validate: %s", strings.Join(problems, "; "))
	}

	defBlob, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("marshal definition: %w", err)
	}
	defDesc := ociDescriptor{MediaType: mediaDefinition, Digest: blobDigest(defBlob), Size: int64(len(defBlob))}

	prov := BundleProvenance{
		ContentDigest: t.ContentDigest(),
		Source:        t.Source,
		Lineage:       t.Lineage,
		Exporter:      BundleExporterVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	cfgBlob, err := json.Marshal(prov)
	if err != nil {
		return nil, fmt.Errorf("marshal provenance: %w", err)
	}
	cfgDesc := ociDescriptor{MediaType: mediaConfig, Digest: blobDigest(cfgBlob), Size: int64(len(cfgBlob))}

	manifest := ociManifest{
		SchemaVersion: 2,
		MediaType:     mediaManifest,
		ArtifactType:  artifactType,
		Config:        cfgDesc,
		Layers:        []ociDescriptor{defDesc},
		Annotations: map[string]string{
			"org.opencontainers.image.title":   t.Name,
			"dev.pi-sandbox.template.digest":   t.ContentDigest(),
			"dev.pi-sandbox.template.exporter": BundleExporterVersion,
		},
	}
	manBlob, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manDesc := ociDescriptor{
		MediaType:   mediaManifest,
		Digest:      blobDigest(manBlob),
		Size:        int64(len(manBlob)),
		Annotations: map[string]string{"org.opencontainers.image.title": t.Name},
	}

	index := ociIndex{SchemaVersion: 2, MediaType: mediaIndex, Manifests: []ociDescriptor{manDesc}}
	idxBlob, _ := json.Marshal(index)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	write := func(name string, data []byte) error {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err := tw.Write(data)
		return err
	}

	if err := write("oci-layout", []byte(`{"imageLayoutVersion":"1.0.0"}`)); err != nil {
		return nil, err
	}
	if err := write("index.json", idxBlob); err != nil {
		return nil, err
	}
	for _, b := range [][]byte{manBlob, cfgBlob, defBlob} {
		if err := write("blobs/sha256/"+strings.TrimPrefix(blobDigest(b), "sha256:"), b); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImportedTemplate is the result of reading a bundle: the definition plus
// its provenance. The template is NOT yet installed.
type ImportedTemplate struct {
	Template   *Template
	Provenance *BundleProvenance
}

// ReadBundle parses an OCI image layout tar and returns the template +
// provenance. It verifies blob digests and re-runs Validate. It does not
// touch the store.
func ReadBundle(tarBytes []byte) (*ImportedTemplate, error) {
	blobs := map[string][]byte{}
	var idxBlob []byte

	tr := tar.NewReader(bytes.NewReader(tarBytes))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read bundle: %w", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		switch {
		case hdr.Name == "index.json":
			idxBlob = data
		case strings.HasPrefix(hdr.Name, "blobs/sha256/"):
			name := path.Base(hdr.Name)
			if blobDigest(data) != "sha256:"+name {
				return nil, fmt.Errorf("blob %s fails digest check", name)
			}
			blobs["sha256:"+name] = data
		}
	}

	if idxBlob == nil {
		return nil, fmt.Errorf("bundle has no index.json")
	}
	var index ociIndex
	if err := json.Unmarshal(idxBlob, &index); err != nil || len(index.Manifests) == 0 {
		return nil, fmt.Errorf("bad index.json")
	}
	manBlob, ok := blobs[index.Manifests[0].Digest]
	if !ok {
		return nil, fmt.Errorf("manifest blob missing")
	}
	var manifest ociManifest
	if err := json.Unmarshal(manBlob, &manifest); err != nil {
		return nil, fmt.Errorf("bad manifest")
	}
	if manifest.ArtifactType != artifactType {
		return nil, fmt.Errorf("not a pi-sandbox template bundle (artifactType %q)", manifest.ArtifactType)
	}
	if len(manifest.Layers) != 1 {
		return nil, fmt.Errorf("expected exactly one definition layer, got %d", len(manifest.Layers))
	}

	defBlob, ok := blobs[manifest.Layers[0].Digest]
	if !ok {
		return nil, fmt.Errorf("definition blob missing")
	}
	var t Template
	if err := yaml.Unmarshal(defBlob, &t); err != nil {
		return nil, fmt.Errorf("parse definition: %w", err)
	}
	if problems := t.Validate(); len(problems) > 0 {
		return nil, fmt.Errorf("bundle template does not validate: %s", strings.Join(problems, "; "))
	}

	var prov BundleProvenance
	if cfgBlob, ok := blobs[manifest.Config.Digest]; ok {
		_ = json.Unmarshal(cfgBlob, &prov)
	}

	return &ImportedTemplate{Template: &t, Provenance: &prov}, nil
}

// Import reads a bundle and installs the template under newName (or the
// bundle's own name when newName is empty), marking it source.type
// imported. Rejects a collision with an existing template.
func (s *Store) Import(tarBytes []byte, newName string) (*Template, error) {
	imp, err := ReadBundle(tarBytes)
	if err != nil {
		return nil, err
	}
	name := newName
	if name == "" {
		name = imp.Template.Name
	}
	if _, err := s.Get(name); err == nil {
		return nil, fmt.Errorf("template %q already exists", name)
	}

	t := *imp.Template
	t.Name = name
	t.Source = &Source{Type: SourceImported}
	if imp.Provenance != nil && imp.Provenance.Source != nil {
		t.Source.Parent = imp.Provenance.Source.Parent
		t.Source.ForkedFrom = imp.Provenance.Source.ForkedFrom
	}
	now := time.Now().UTC().Format(time.RFC3339)
	t.CreatedAt, t.UpdatedAt = now, now
	t.Lineage = &Lineage{Generation: 1}
	t.Lineage.ContentDigest = t.ContentDigest()

	if err := s.Create(name, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
