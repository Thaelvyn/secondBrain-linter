// Package vault scans a secondBrain vault directory into an in-memory
// representation suitable for linting.
package vault

import (
	"bufio"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// File describes a single regular file found in the vault.
type File struct {
	Path        string // path relative to the vault root, forward slashes
	Name        string // base name
	Content     string // full content for markdown files
	HasFM       bool   // frontmatter was present and parsed
	Frontmatter map[string]any
	Markdown    bool
}

// Vault is the result of a single walk over the vault root.
type Vault struct {
	Root   string // cleaned root path
	Files  []*File
	ByPath map[string]*File
	Dirs   []string // relative dir paths, forward slashes
	DirSet map[string]bool
}

// Scan walks root once and loads every file. Dot-directories (e.g. .obsidian,
// .trash) and dot-files are skipped entirely: they are exempt from lint rules
// and can be large. Markdown files are read fully (needed for wikilink and
// embed analysis) and their YAML frontmatter is parsed lazily here.
func Scan(root string) (*Vault, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &notDirError{root}
	}

	root = filepath.Clean(root)
	v := &Vault{
		Root:   root,
		ByPath: map[string]*File{},
		DirSet: map[string]bool{},
	}

	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return rerr
			}
			if rel != "." {
				s := filepath.ToSlash(rel)
				v.Dirs = append(v.Dirs, s)
				v.DirSet[s] = true
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		rels := filepath.ToSlash(rel)
		f := &File{Path: rels, Name: d.Name(), Markdown: strings.HasSuffix(d.Name(), ".md")}
		if f.Markdown {
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			f.Content = string(data)
			f.HasFM, f.Frontmatter = parseFrontmatter(data)
		}
		v.Files = append(v.Files, f)
		v.ByPath[rels] = f
		return nil
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

type notDirError struct{ path string }

func (e *notDirError) Error() string { return e.path + ": not a directory" }

// parseFrontmatter extracts the leading YAML block delimited by --- lines.
// Malformed YAML yields (false, nil): the file is treated as having no
// frontmatter rather than aborting the scan.
func parseFrontmatter(data []byte) (bool, map[string]any) {
	if len(data) < 4 || !bytes.HasPrefix(data, []byte("---\n")) {
		return false, nil
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	if !sc.Scan() {
		return false, nil
	}
	var buf []byte
	for sc.Scan() {
		if sc.Text() == "---" {
			var m map[string]any
			if err := yaml.Unmarshal(buf, &m); err != nil {
				return false, nil
			}
			return true, m
		}
		buf = append(buf, sc.Bytes()...)
		buf = append(buf, '\n')
	}
	return false, nil
}
