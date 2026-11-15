package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

func TestMergeConfig(t *testing.T) {
	fileCfg := config.Config{
		APIProvider: config.ProviderWttr,
		APIKey:      "filekey",
		City:        "Moscow",
		Unit:        config.UnitMetric,
		Interval:    time.Minute,
	}
	cliCfg := config.Config{
		City:     "Berlin",
		Unit:     config.UnitImperial,
		Fancy:    true,
		Forecast: 3,
		Hourly:   true,
		NoCache:  true,
		Format:   "{temp}",
	}
	got := config.MergeConfig(fileCfg, cliCfg)

	if got.City != "Berlin" {
		t.Errorf("City = %q, want CLI override %q", got.City, "Berlin")
	}
	if got.Unit != config.UnitImperial {
		t.Errorf("Unit = %q, want %q", got.Unit, config.UnitImperial)
	}
	if got.APIKey != "filekey" {
		t.Errorf("APIKey = %q, want file value preserved", got.APIKey)
	}
	if !got.Fancy {
		t.Error("Fancy = false, want true from CLI")
	}
	if got.Forecast != 3 {
		t.Errorf("Forecast = %d, want 3", got.Forecast)
	}
	if got.Interval != time.Minute {
		t.Errorf("Interval = %v, want file value preserved", got.Interval)
	}
	if !got.Hourly || !got.NoCache {
		t.Errorf("Hourly=%v NoCache=%v, want both true from CLI", got.Hourly, got.NoCache)
	}
	if got.Format != "{temp}" {
		t.Errorf("Format = %q, want CLI override", got.Format)
	}
}

func TestReadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wrep")
	content := `# comment line
apiKey = secret
defaultCity=Oslo

units=imperial
apiProvider=weatherapi
fancy=on
quiet=yes
hourly=on
cache=off
cacheTTL=5m
format={emoji} {temp}
interval=90s
not a key value line
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.ReadConfigFile(path)
	if err != nil {
		t.Fatalf("ReadConfigFile: %v", err)
	}
	if cfg.APIKey != "secret" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "secret")
	}
	if cfg.City != "Oslo" {
		t.Errorf("City = %q, want %q", cfg.City, "Oslo")
	}
	if cfg.Unit != config.UnitImperial {
		t.Errorf("Unit = %q, want %q", cfg.Unit, config.UnitImperial)
	}
	if cfg.APIProvider != config.ProviderWeatherAPI {
		t.Errorf("APIProvider = %q, want %q", cfg.APIProvider, config.ProviderWeatherAPI)
	}
	if !cfg.Fancy || !cfg.Quiet {
		t.Errorf("Fancy=%v Quiet=%v, want both true", cfg.Fancy, cfg.Quiet)
	}
	if cfg.Interval != 90*time.Second {
		t.Errorf("Interval = %v, want 90s", cfg.Interval)
	}
	if !cfg.Hourly {
		t.Error("Hourly = false, want true")
	}
	if !cfg.NoCache {
		t.Error("cache=off should set NoCache")
	}
	if cfg.CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want 5m", cfg.CacheTTL)
	}
	if cfg.Format != "{emoji} {temp}" {
		t.Errorf("Format = %q, want {emoji} {temp}", cfg.Format)
	}
}

func TestReadConfigFileInvalidInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wrep")
	if err := os.WriteFile(path, []byte("interval=banana\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.ReadConfigFile(path); err == nil {
		t.Error("expected error for invalid interval, got nil")
	}
}

func TestReadConfigFileMissing(t *testing.T) {
	if _, err := config.ReadConfigFile(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wrep")
	if err := config.GenerateDefaultConfig(path); err != nil {
		t.Fatalf("GenerateDefaultConfig: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("config file mode = %o, want 0600 (may contain API key)", perm)
		}
	}

	cfg, err := config.ReadConfigFile(path)
	if err != nil {
		t.Fatalf("generated config unreadable: %v", err)
	}
	if cfg.City == "" {
		t.Error("generated config has no defaultCity")
	}
	if cfg.APIProvider != config.ProviderWttr {
		t.Errorf("generated APIProvider = %q, want %q", cfg.APIProvider, config.ProviderWttr)
	}
}

func TestParseBool(t *testing.T) {
	trues := []string{"on", "true", "yes", "1", " ON ", "True"}
	falses := []string{"off", "false", "no", "0", "", "banana"}
	for _, s := range trues {
		if !config.ParseBool(s) {
			t.Errorf("ParseBool(%q) = false, want true", s)
		}
	}
	for _, s := range falses {
		if config.ParseBool(s) {
			t.Errorf("ParseBool(%q) = true, want false", s)
		}
	}
}

func TestValidProviderAndUnit(t *testing.T) {
	if !config.ValidProvider(config.ProviderWttr) || !config.ValidProvider(config.ProviderWeatherAPI) || !config.ValidProvider(config.ProviderOpenMeteo) {
		t.Error("known providers should be valid")
	}
	if config.ValidProvider("openweathermap") {
		t.Error("unknown provider should be invalid")
	}
	if !config.ValidUnit(config.UnitMetric) || !config.ValidUnit(config.UnitImperial) {
		t.Error("known units should be valid")
	}
	if config.ValidUnit("kelvin") {
		t.Error("unknown unit should be invalid")
	}
}
