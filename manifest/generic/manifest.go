package generic

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opencontainers/go-digest"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
)

const (
	// GenericMediaTypePrefix is the prefix for generic manifests
	GenericMediaTypePrefix = "x-application/"
)

func init() {
	genericFunc := func(b []byte) (distribution.Manifest, distribution.Descriptor, error) {
		m := new(DeserializedManifest)
		err := m.UnmarshalJSON(b)
		if err != nil {
			return nil, distribution.Descriptor{}, err
		}
		dgst := digest.FromBytes(b)
		return m, distribution.Descriptor{Digest: dgst, Size: int64(len(b)), MediaType: m.MediaType}, err
	}
	err := distribution.RegisterGenericManifestSchema(GenericMediaTypePrefix, genericFunc)
	if err != nil {
		panic(fmt.Sprintf("Unable to register manifest: %s", err))
	}
}

// ensure that DeserializedManifest implemets distribution.Manifest
var _ distribution.Manifest = &DeserializedManifest{}

// Manifest is a generic manifest that might contain references
type Manifest struct {
	manifest.Versioned
	Refs []distribution.Descriptor `json:"references"`
}

// References returnes the descriptors of this manifests references.
func (m Manifest) References() []distribution.Descriptor {
	return m.Refs
}

// DeserializedManifest wraps Manifest with a copy of the original JSON.
// It satisfies the distribution.Manifest interface.
// Original Json could be used to unmarshall a more complete manifest data type
type DeserializedManifest struct {
	Manifest
	canonical []byte
}

// FromStruct takes a Manifest structure, marshals it to JSON, and returns a
// DeserializedManifest which contains the manifest and its JSON representation.
func FromStruct(m interface{}) (*DeserializedManifest, error) {
	var deserialized DeserializedManifest

	var err error
	deserialized.canonical, err = json.MarshalIndent(&m, "", "   ")
	if err != nil {
		return nil, err
	}
	// unmarshall partially in Manifest
	err = json.Unmarshal(deserialized.canonical, &deserialized.Manifest)
	return &deserialized, err
}

// Payload returns the raw content of the manifest. The contents can be used to
// calculate the content identifier.
func (m DeserializedManifest) Payload() (string, []byte, error) {
	return m.MediaType, m.canonical, nil
}

// UnmarshalJSON populates a new Manifest struct from JSON data.
func (m *DeserializedManifest) UnmarshalJSON(b []byte) error {
	m.canonical = make([]byte, len(b), len(b))
	// store manifest in canonical
	copy(m.canonical, b)

	// Unmarshal canonical JSON into Manifest object
	var manifest Manifest
	if err := json.Unmarshal(m.canonical, &manifest); err != nil {
		return err
	}

	if !strings.HasPrefix(manifest.MediaType, GenericMediaTypePrefix) {
		return fmt.Errorf("mediaType in manifest should have prefix '%s', '%s' has not",
			GenericMediaTypePrefix, manifest.MediaType)

	}

	m.Manifest = manifest

	return nil
}

// MarshalJSON returns the contents of canonical. If canonical is empty,
// marshals the inner contents.
func (m *DeserializedManifest) MarshalJSON() ([]byte, error) {
	if len(m.canonical) > 0 {
		return m.canonical, nil
	}

	return nil, errors.New("JSON representation not initialized in DeserializedManifest")
}
