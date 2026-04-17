package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repo     = "booscaaa/estrova"
	apiBase  = "https://api.github.com/repos/" + repo
	rawBase  = "https://raw.githubusercontent.com/" + repo + "/main/skills/estrova"
)

var skills = []string{"estrova-coach", "estrova-analysis", "estrova-plan", "estrova-resolve-conflicts"}

func Run() error {
	fmt.Println("Verificando última versão...")

	tag, err := latestTag()
	if err != nil {
		return fmt.Errorf("erro ao buscar versão: %w", err)
	}
	fmt.Printf("Última versão: %s\n", tag)

	if err := updateBinary(tag); err != nil {
		return fmt.Errorf("erro ao atualizar binário: %w", err)
	}

	if err := updateSkills(); err != nil {
		fmt.Printf("⚠️  Skills não atualizadas: %v\n", err)
	}

	fmt.Println("\n✅ Estrova atualizado com sucesso!")
	fmt.Println("Reinicie o Claude Code para aplicar as mudanças.")
	return nil
}

func latestTag() (string, error) {
	resp, err := http.Get(apiBase + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("nenhuma release encontrada")
	}
	return release.TagName, nil
}

func updateBinary(tag string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var filename string
	if goos == "windows" {
		filename = fmt.Sprintf("estrova_%s_%s.zip", goos, goarch)
	} else {
		filename = fmt.Sprintf("estrova_%s_%s.tar.gz", goos, goarch)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, filename)
	fmt.Printf("Baixando %s...\n", filename)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download falhou: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var binData []byte
	if goos == "windows" {
		binData, err = extractZip(data, "estrova.exe")
	} else {
		binData, err = extractTarGz(data, "estrova")
	}
	if err != nil {
		return fmt.Errorf("erro ao extrair binário: %w", err)
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return err
	}

	tmp := self + ".tmp"
	if err := os.WriteFile(tmp, binData, 0755); err != nil {
		return fmt.Errorf("sem permissão para escrever em %s (tente com sudo)", filepath.Dir(self))
	}

	if err := os.Rename(tmp, self); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("erro ao substituir binário (tente com sudo): %w", err)
	}

	fmt.Printf("✅ Binário atualizado: %s\n", self)
	return nil
}

func updateSkills() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	pluginDir := filepath.Join(home, ".claude", "plugins", "local", "plugins", "estrova")
	skillsDir := filepath.Join(pluginDir, "skills")

	metaDir := filepath.Join(pluginDir, ".claude-plugin")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}
	pluginJSON := []byte(`{
  "name": "estrova",
  "version": "1.0.0",
  "description": "Personal Strava training coach — training plans, performance analysis and conflict resolution"
}`)
	if err := os.WriteFile(filepath.Join(metaDir, "plugin.json"), pluginJSON, 0644); err != nil {
		return err
	}

	fmt.Println("Atualizando skills...")
	for _, skill := range skills {
		dir := filepath.Join(skillsDir, skill)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		url := fmt.Sprintf("%s/%s/SKILL.md", rawBase, skill)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("skill %s: HTTP %d", skill, resp.StatusCode)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		dest := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s\n", skill)
	}

	if err := updateMarketplace(home); err != nil {
		return fmt.Errorf("marketplace: %w", err)
	}
	return updateSettings(home)
}

func updateMarketplace(home string) error {
	marketplacePath := filepath.Join(home, ".claude", "plugins", "local", ".claude-plugin", "marketplace.json")

	data, err := os.ReadFile(marketplacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return writeDefaultMarketplace(marketplacePath, home)
		}
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	plugins, _ := m["plugins"].([]interface{})
	for _, p := range plugins {
		if pm, ok := p.(map[string]interface{}); ok {
			if pm["name"] == "estrova" {
				return nil
			}
		}
	}

	m["plugins"] = append(plugins, map[string]interface{}{
		"name":        "estrova",
		"description": "Personal Strava training coach — training plans, performance analysis and conflict resolution",
		"version":     "1.0.0",
		"author":      map[string]interface{}{"name": "booscaaa"},
		"source":      "./plugins/estrova",
		"category":    "health",
	})

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(marketplacePath, out, 0644)
}

func writeDefaultMarketplace(path, home string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	m := map[string]interface{}{
		"$schema":     "https://anthropic.com/claude-code/marketplace.schema.json",
		"name":        "local",
		"description": "Local plugins for personal use",
		"owner":       map[string]interface{}{"name": "booscaaa"},
		"plugins": []interface{}{
			map[string]interface{}{
				"name":        "estrova",
				"description": "Personal Strava training coach — training plans, performance analysis and conflict resolution",
				"version":     "1.0.0",
				"author":      map[string]interface{}{"name": "booscaaa"},
				"source":      "./plugins/estrova",
				"category":    "health",
			},
		},
	}
	out, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(path, out, 0644)
}

func updateSettings(home string) error {
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return writeDefaultSettings(settingsPath, home)
		}
		return err
	}

	var s map[string]interface{}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	plugins, _ := s["enabledPlugins"].(map[string]interface{})
	if plugins == nil {
		plugins = map[string]interface{}{}
	}
	if plugins["estrova@local"] == true {
		return nil
	}
	plugins["estrova@local"] = true
	s["enabledPlugins"] = plugins

	if _, ok := s["extraKnownMarketplaces"]; !ok {
		s["extraKnownMarketplaces"] = map[string]interface{}{
			"local": map[string]interface{}{
				"source": map[string]interface{}{
					"source": "directory",
					"path":   filepath.Join(home, ".claude", "plugins", "local"),
				},
			},
		}
	}

	out, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0644)
}

func writeDefaultSettings(path, home string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	s := map[string]interface{}{
		"enabledPlugins": map[string]interface{}{"estrova@local": true},
		"extraKnownMarketplaces": map[string]interface{}{
			"local": map[string]interface{}{
				"source": map[string]interface{}{
					"source": "directory",
					"path":   filepath.Join(home, ".claude", "plugins", "local"),
				},
			},
		},
	}
	out, _ := json.MarshalIndent(s, "", "    ")
	return os.WriteFile(path, out, 0644)
}

func extractTarGz(data []byte, target string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.TrimPrefix(hdr.Name, "./") == target {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("arquivo %q não encontrado no tarball", target)
}

func extractZip(data []byte, target string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if f.Name == target {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("arquivo %q não encontrado no zip", target)
}
