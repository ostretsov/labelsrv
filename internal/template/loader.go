package template

import (
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// TemplateLoader indexes template files by name and reads them from disk on every Get.
type TemplateLoader struct {
	dir      string
	files    map[string]string    // template name → file path (empty string for in-memory)
	inMemory map[string]*Template // in-memory templates injected via Register (tests only)
	mu       sync.RWMutex
}

// NewTemplateLoader creates a new TemplateLoader.
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{
		files:    make(map[string]string),
		inMemory: make(map[string]*Template),
	}
}

// LoadAll scans dir and indexes all valid .yaml/.yml/.json template files.
func (l *TemplateLoader) LoadAll(dir string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.dir = dir
	l.files = make(map[string]string)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(dir, name)

		t, err := ParseFile(path)
		if err != nil {
			log.Printf("warning: failed to load template %q: %v", path, err)
			continue
		}

		if err := Validate(t); err != nil {
			log.Printf("warning: template %q validation failed: %v", path, err)
			continue
		}

		l.files[t.Name] = path
		log.Printf("indexed template %q from %s", t.Name, path)
	}

	return nil
}

// Get reads and parses the template from disk on every call.
func (l *TemplateLoader) Get(name string) (*Template, bool) {
	l.mu.RLock()
	path, ok := l.files[name]
	mem := l.inMemory[name]
	l.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if path == "" {
		return mem, mem != nil
	}

	t, err := ParseFile(path)
	if err != nil {
		log.Printf("error reading template %q: %v", path, err)
		return nil, false
	}

	return t, true
}

// List returns all indexed template names.
func (l *TemplateLoader) List() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.files))
	for name := range l.files {
		names = append(names, name)
	}

	return names
}

// Register adds a template path directly to the index (useful for testing).
func (l *TemplateLoader) Register(name string, t *Template) {
	// Tests inject in-memory templates; store a sentinel so Get can return them.
	l.mu.Lock()
	defer l.mu.Unlock()

	l.files[name] = ""
	l.inMemory[name] = t
}

// ForTest is an alias for Register used in tests.
func (l *TemplateLoader) ForTest(name string, t *Template) {
	l.Register(name, t)
}

// All reads and parses every indexed template from disk.
func (l *TemplateLoader) All() map[string]*Template {
	l.mu.RLock()

	paths := make(map[string]string, len(l.files))
	maps.Copy(paths, l.files)

	l.mu.RUnlock()

	result := make(map[string]*Template, len(paths))

	for name, path := range paths {
		if path == "" {
			if t, ok := l.inMemory[name]; ok {
				result[name] = t
			}

			continue
		}

		t, err := ParseFile(path)
		if err != nil {
			log.Printf("error reading template %q for All(): %v", path, err)
			continue
		}

		result[name] = t
	}

	return result
}

// Watch watches the directory for file changes and updates the index.
func (l *TemplateLoader) Watch(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("error creating watcher: %v", err)
		return
	}

	if err := watcher.Add(dir); err != nil {
		log.Printf("error watching directory %q: %v", dir, err)

		_ = watcher.Close()

		return
	}

	go func() {
		defer func() { _ = watcher.Close() }()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext != ".yaml" && ext != ".yml" && ext != ".json" {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
					fmt.Printf("[dev] file changed: %s, updating index...\n", event.Name)

					if err := l.reloadFile(event.Name, event.Op); err != nil {
						log.Printf("error updating index for %q: %v", event.Name, err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Printf("watcher error: %v", err)
			}
		}
	}()
}

func (l *TemplateLoader) reloadFile(path string, op fsnotify.Op) error {
	if op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		l.mu.Lock()
		defer l.mu.Unlock()

		for name, p := range l.files {
			if p == path {
				delete(l.files, name)
				fmt.Printf("[dev] removed template %q from index\n", name)

				break
			}
		}

		return nil
	}

	t, err := ParseFile(path)
	if err != nil {
		return fmt.Errorf("parsing %q: %w", path, err)
	}

	if err := Validate(t); err != nil {
		return fmt.Errorf("validating %q: %w", path, err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.files[t.Name] = path
	fmt.Printf("[dev] indexed template %q\n", t.Name)

	return nil
}
