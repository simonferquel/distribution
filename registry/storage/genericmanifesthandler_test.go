package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/docker/distribution/manifest/generic"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/storage/driver/inmemory"
)

type testGenericManifest struct {
	generic.Manifest
	CustomField string `json:"custom_field"`
}

func TestGenericManifestWithCustomData(t *testing.T) {
	ctx := context.Background()
	inmemoryDriver := inmemory.New()
	registry := createRegistry(t, inmemoryDriver)
	repo := makeRepository(t, registry, "test")
	manifestService := makeManifestService(t, repo)

	testData := &testGenericManifest{}
	testData.SchemaVersion = 2
	testData.MediaType = generic.GenericMediaTypePrefix + "test-data"
	testData.CustomField = "test"

	dm, err := generic.FromStruct(testData)
	if err != nil {
		t.Fatal(err)
	}
	dgst, err := manifestService.Put(ctx, dm)
	if err != nil {
		t.Fatal(err)
	}

	stored, err := manifestService.Get(ctx, dgst)
	if err != nil {
		t.Fatal(err)
	}

	var resultData testGenericManifest
	resultMediaType, resultPayload, err := stored.Payload()
	if err != nil {
		t.Fatal(err)
	}
	if resultMediaType != testData.MediaType {
		t.Errorf("Expected mediatype %q, got %q", testData.MediaType, resultMediaType)
	}
	if err := json.Unmarshal(resultPayload, &resultData); err != nil {
		t.Fatal(err)
	}
	if resultData.CustomField != "test" {
		t.Fatalf("Failed to store custom data: Expected %q, got %q", "test", resultData.CustomField)
	}
}

func TestGenericManifestWithReferences(t *testing.T) {
	ctx := context.Background()
	inmemoryDriver := inmemory.New()
	registry := createRegistry(t, inmemoryDriver)
	repo := makeRepository(t, registry, "test")
	manifestService := makeManifestService(t, repo)
	blobsService := repo.Blobs(ctx)
	layer, err := blobsService.Put(ctx, schema2.MediaTypeLayer, nil)
	if err != nil {
		t.Fatal(err)
	}
	testData := &testGenericManifest{}
	testData.SchemaVersion = 2
	testData.MediaType = generic.GenericMediaTypePrefix + "test-data"
	testData.Refs = append(testData.Refs, layer)

	dm, err := generic.FromStruct(testData)
	if err != nil {
		t.Fatal(err)
	}
	dgst, err := manifestService.Put(ctx, dm)
	if err != nil {
		t.Fatal(err)
	}

	// remove both manifest and layer
	if err := manifestService.Delete(ctx, dgst); err != nil {
		t.Fatal(err)
	}
	if err := blobsService.Delete(ctx, layer.Digest); err != nil {
		t.Fatal(err)
	}

	// try to re-put the manifest (should fail)
	_, err = manifestService.Put(ctx, dm)
	if err == nil {
		t.Fatal("Manifest with inexisting reference should fail on storage")
	}

}
