package luaprovider

import (
	"context"
	"strings"
	"testing"

	"github.com/luevano/libmangal"
)

func settingsLoader(t *testing.T, script string, options Options) libmangal.ProviderLoader {
	t.Helper()
	info := libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0", Settings: []libmangal.ProviderSetting{
		{Key: "quality", Kind: libmangal.SettingEnum, Default: "high", Values: []string{"low", "high"}},
		{Key: "data_saver", Kind: libmangal.SettingBool, Default: "true"},
		{Key: "pages", Kind: libmangal.SettingInt, Default: "3"},
		{Key: "groups", Kind: libmangal.SettingList},
	}}
	loader, err := NewLoader([]byte(script+testScript), info, options)
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

func TestScriptSettings(t *testing.T) {
	const script = `
local settings = require("sdk").settings
assert(settings.get("quality") == "low", "quality")
assert(settings.get("data_saver") == true, "data_saver")
assert(settings.get("pages") == 3, "pages")
local groups = settings.get("groups")
assert(#groups == 2 and groups[1] == "a" and groups[2] == "b", "groups")
assert(not pcall(settings.get, "nope"), "undeclared key")
`
	options := DefaultOptions()
	options.Settings = func(id string) map[string]string {
		if id != "test" {
			t.Errorf("asked for %q", id)
		}
		return map[string]string{"quality": "low", "groups": "a, b"}
	}
	p, err := settingsLoader(t, script, options).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p.Close()

	// no hook: the defaults
	p, err = settingsLoader(t, `assert(require("sdk").settings.get("quality") == "high", "default")`, DefaultOptions()).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p.Close()

	options.Settings = func(string) map[string]string { return map[string]string{"quality": "ultra"} }
	if _, err := settingsLoader(t, "", options).Load(context.Background()); err == nil || !strings.Contains(err.Error(), "quality") {
		t.Errorf("invalid value: %v", err)
	}
}
