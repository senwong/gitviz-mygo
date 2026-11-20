package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func configFields(config *RepoConfig) map[string]reflect.Value {
	result := map[string]reflect.Value{}
	value := reflect.ValueOf(config).Elem()
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		key := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if key != "extra" {
			result[key] = value.Field(i)
		}
	}
	return result
}

func parseConfig(text string) RepoConfig {
	config := defaultConfig()
	config.Extra = map[string]string{}
	fields := configFields(&config)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		field, ok := fields[key]
		if !ok {
			config.Extra[key] = value
			continue
		}
		switch field.Kind() {
		case reflect.Bool:
			if v, err := strconv.ParseBool(value); err == nil {
				field.SetBool(v)
			}
		case reflect.Int:
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(v)
			}
		case reflect.Float64:
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				field.SetFloat(v)
			}
		case reflect.String:
			field.SetString(value)
		case reflect.Slice:
			field.Set(reflect.ValueOf(splitConfigList(value)))
		}
	}
	config.normalize()
	return config
}

func splitConfigList(value string) []string {
	var result []string
	for _, v := range strings.FieldsFunc(value, func(r rune) bool { return r == ';' || r == ',' }) {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}

func (config *RepoConfig) normalize() {
	config.DateWidth = bounded(config.DateWidth, 70, 600, 150)
	config.AuthorWidth = bounded(config.AuthorWidth, 60, 600, 130)
	config.CommitWidth = bounded(config.CommitWidth, 65, 300, 80)
	if config.GraphStyle != "angular" {
		config.GraphStyle = "rounded"
	}
	if config.RepoOrder != "path" && config.RepoOrder != "given" {
		config.RepoOrder = "name"
	}
	if config.ColorPreset < 0 || config.ColorPreset > 2 {
		config.ColorPreset = 0
	}
}

func bounded(value, lo, hi, fallback float64) float64 {
	if value != value {
		return fallback
	}
	return min(hi, max(lo, value))
}

func serializeConfig(config RepoConfig) string {
	values := map[string]string{}
	fields := configFields(&config)
	for k, v := range config.Extra {
		values[k] = v
	}
	for k, v := range fields {
		if v.Kind() == reflect.Slice {
			values[k] = strings.Join(v.Interface().([]string), ";")
		} else {
			values[k] = fmt.Sprint(v.Interface())
		}
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("# gitviz repository configuration\n")
	for _, key := range keys {
		out.WriteString(key + "=" + values[key] + "\n")
	}
	return out.String()
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gitviz-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func (a *App) configForRepo(repo string) RepoConfig {
	if cfg, ok := a.prefs.Configs[repo]; ok {
		cfg.normalize()
		return cfg
	}
	if data, err := os.ReadFile(filepath.Join(repo, ".gitviz.conf")); err == nil {
		return parseConfig(string(data))
	}
	cfg := defaultConfig()
	cfg.Remotes = a.prefs.Remotes
	cfg.ShowDetail = a.prefs.Detail
	return cfg
}

func (a *App) saveConfig(reload bool) {
	a.config.normalize()
	if a.prefs.Configs == nil {
		a.prefs.Configs = map[string]RepoConfig{}
	}
	if repo := a.repo(); repo != "" {
		a.prefs.Configs[repo] = a.config
	}
	a.rebuildFileRows()
	a.detailVisible = a.config.ShowDetail
	a.prefs.Remotes = a.config.Remotes
	a.persist()
	if reload {
		a.load(true)
	} else {
		a.rebuildVisible()
	}
}

func (a *App) exportConfig() {
	if a.repo() == "" {
		return
	}
	a.fail(atomicWrite(filepath.Join(a.repo(), ".gitviz.conf"), []byte(serializeConfig(a.config))))
}

func (a *App) importConfig() {
	data, err := os.ReadFile(filepath.Join(a.repo(), ".gitviz.conf"))
	if err != nil {
		a.fail(err)
		return
	}
	a.config = parseConfig(string(data))
	a.saveConfig(true)
}

func cloneConfig(config RepoConfig) RepoConfig {
	data, _ := json.Marshal(config)
	var copy RepoConfig
	_ = json.Unmarshal(data, &copy)
	return copy
}
