package option

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultDataPath(t *testing.T) {
	t.Setenv("TG_BOT_TOKEN", "token")
	t.Setenv("DATA_PATH", "")
	t.Setenv("SUBSCRIPTION_BOT_DATA", "")
	t.Setenv("TZ", "UTC")

	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("data", "subscriptions.db")
	if config.DataPath != want {
		t.Fatalf("unexpected data path: %q", config.DataPath)
	}
}
