package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
)

// LoadAll scans pluginsDir and loads all valid plugins
func (pm *PluginManager) LoadAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if err := os.MkdirAll(pm.pluginsDir, 0755); err != nil {
		return fmt.Errorf("cannot create plugins dir: %w", err)
	}

	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		return fmt.Errorf("cannot read plugins dir: %w", err)
	}

	pm.plugins = make(map[string]*Plugin)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(pm.pluginsDir, entry.Name())
		manifest, err := loadManifest(dir)
		if err != nil {
			gologger.Warning().Msgf("plugin: skip %s: %v", entry.Name(), err)
			continue
		}

		p := &Plugin{
			Manifest: manifest,
			Dir:      dir,
			Enabled:  isEnabled(dir),
			Config:   loadPluginConfig(dir),
		}
		pm.plugins[manifest.Name] = p
		gologger.Info().Msgf("plugin: loaded %s v%s (enabled=%v)", manifest.Name, manifest.Version, p.Enabled)
	}

	return nil
}

// GetTemplatePaths returns absolute template directory paths from all enabled plugins
func (pm *PluginManager) GetTemplatePaths() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var paths []string
	for _, p := range pm.plugins {
		if !p.Enabled {
			continue
		}
		for _, pattern := range p.Manifest.Capabilities.Templates {
			matches, err := filepath.Glob(filepath.Join(p.Dir, pattern))
			if err != nil {
				continue
			}
			for _, m := range matches {
				info, err := os.Stat(m)
				if err == nil && info.IsDir() {
					paths = append(paths, m)
				}
			}
		}
	}
	return paths
}

// GetUserdataPaths returns absolute userdata file paths from all enabled plugins
func (pm *PluginManager) GetUserdataPaths() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var paths []string
	for _, p := range pm.plugins {
		if !p.Enabled {
			continue
		}
		for _, pattern := range p.Manifest.Capabilities.Userdata {
			matches, err := filepath.Glob(filepath.Join(p.Dir, pattern))
			if err != nil {
				continue
			}
			paths = append(paths, matches...)
		}
	}
	return paths
}

// GetHooks returns all hook scripts for a given hook point from enabled plugins
func (pm *PluginManager) GetHooks(hookPoint string) []HookEntry {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var entries []HookEntry
	for _, p := range pm.plugins {
		if !p.Enabled {
			continue
		}
		scriptRel, ok := p.Manifest.Capabilities.Hooks[hookPoint]
		if !ok {
			continue
		}
		scriptPath := filepath.Join(p.Dir, scriptRel)
		if _, err := os.Stat(scriptPath); err != nil {
			continue
		}
		entries = append(entries, HookEntry{
			PluginName: p.Manifest.Name,
			ScriptPath: scriptPath,
			PluginDir:  p.Dir,
			Config:     p.Config,
		})
	}
	return entries
}
