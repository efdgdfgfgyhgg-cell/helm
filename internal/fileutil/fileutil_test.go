/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fileutil

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAtomicWriteFile tests the happy path of AtomicWriteFile function.
// It verifies that the function correctly writes content to a file with the specified mode.
func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()

	testpath := filepath.Join(dir, "test")
	stringContent := "Test content"
	reader := bytes.NewReader([]byte(stringContent))
	mode := os.FileMode(0o644)

	require.NoError(t, AtomicWriteFile(testpath, reader, mode))

	got, err := os.ReadFile(testpath)
	require.NoError(t, err)

	require.Equal(t, stringContent, string(got))

	gotinfo, err := os.Stat(testpath)
	require.NoError(t, err)

	require.Equal(t, mode, gotinfo.Mode())
}

// TestAtomicWriteFile_CreateTempError tests the error path when os.CreateTemp fails
func TestAtomicWriteFile_CreateTempError(t *testing.T) {
	invalidPath := "/invalid/path/that/does/not/exist/testfile"

	reader := bytes.NewReader([]byte("test content"))
	mode := os.FileMode(0o644)

	assert.Error(t, AtomicWriteFile(invalidPath, reader, mode), "Expected error when CreateTemp fails")
}

// TestAtomicWriteFile_EmptyContent tests with empty content
func TestAtomicWriteFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	testpath := filepath.Join(dir, "empty_helm")

	reader := bytes.NewReader([]byte(""))
	mode := os.FileMode(0o644)

	require.NoError(t, AtomicWriteFile(testpath, reader, mode), "AtomicWriteFile error with empty content")

	got, err := os.ReadFile(testpath)
	require.NoError(t, err)

	require.Empty(t, got)
}

// TestAtomicWriteFile_LargeContent tests with large content
func TestAtomicWriteFile_LargeContent(t *testing.T) {
	dir := t.TempDir()
	testpath := filepath.Join(dir, "large_test")

	// Create a large content string
	largeContent := strings.Repeat("HELM", 1024*1024)
	reader := bytes.NewReader([]byte(largeContent))
	mode := os.FileMode(0o644)

	require.NoError(t, AtomicWriteFile(testpath, reader, mode), "AtomicWriteFile error with large content")

	got, err := os.ReadFile(testpath)
	require.NoError(t, err)

	require.Equal(t, largeContent, string(got))
}

// TestPlatformAtomicWriteFile_OverwritesExisting verifies that the platform
// helper replaces existing files instead of silently skipping them.
func TestPlatformAtomicWriteFile_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite_test")

	first := bytes.NewReader([]byte("first"))
	require.NoError(t, PlatformAtomicWriteFile(path, first, 0o644), "first write failed")

	second := bytes.NewReader([]byte("second"))
	require.NoError(t, PlatformAtomicWriteFile(path, second, 0o644), "second write failed")

	contents, err := os.ReadFile(path)
	require.NoError(t, err, "failed reading result")

	require.Equal(t, "second", string(contents))
}

func TestAtomicWriteFile_ReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repositories.yaml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))
	readErr := errors.New("read failed")
	reader := io.MultiReader(strings.NewReader("partial"), failingReader{readErr})

	require.ErrorIs(t, AtomicWriteFile(path, reader, 0o600), readErr)
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(contents))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "failed writes must not leave temporary files")
	assert.Equal(t, "repositories.yaml", entries[0].Name())
}

func TestAtomicWriteFile_RenameError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "destination")
	require.NoError(t, os.Mkdir(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "original"), []byte("original"), 0o600))

	require.Error(t, AtomicWriteFile(path, strings.NewReader("replacement"), 0o600))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "failed renames must not leave temporary files")
	assert.Equal(t, "destination", entries[0].Name())
	contents, err := os.ReadFile(filepath.Join(path, "original"))
	require.NoError(t, err)
	assert.Equal(t, "original", string(contents))
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}
