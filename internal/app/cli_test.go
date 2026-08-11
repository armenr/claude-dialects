package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestFindShellAliasPrefersZsh(t *testing.T) {
	var commands []string
	alias, shell, config, found := findShellAlias("cc-test",
		func(name string) (string, error) { return "/bin/" + name, nil },
		func(path, command string) ([]byte, error) {
			commands = append(commands, path+" "+command)
			return []byte("cc-test='first'\n"), nil
		})
	if !found || alias != "cc-test='first'" || shell != "zsh" || config != "~/.zshrc" {
		t.Fatalf("findShellAlias() = %q, %q, %q, %v", alias, shell, config, found)
	}
	if !reflect.DeepEqual(commands, []string{"/bin/zsh alias cc-test"}) {
		t.Fatalf("commands = %v", commands)
	}
}

func TestFindShellAliasFallsBackToBash(t *testing.T) {
	alias, shell, config, found := findShellAlias("cc-test",
		func(name string) (string, error) {
			if name == "zsh" {
				return "", errors.New("not found")
			}
			return "/bin/bash", nil
		},
		func(path, command string) ([]byte, error) {
			if path != "/bin/bash" || command != "alias cc-test" {
				t.Fatalf("run(%q, %q)", path, command)
			}
			return []byte("cc-test='linux'\n"), nil
		})
	if !found || alias != "cc-test='linux'" || shell != "bash" || config != "~/.bashrc" {
		t.Fatalf("findShellAlias() = %q, %q, %q, %v", alias, shell, config, found)
	}
}

func TestFindShellAliasHandlesMissingAndInvalidAliases(t *testing.T) {
	lookups := 0
	lookPath := func(string) (string, error) {
		lookups++
		return "/bin/shell", nil
	}
	run := func(string, string) ([]byte, error) { return nil, errors.New("alias not found") }
	if _, _, _, found := findShellAlias("cc-test", lookPath, run); found {
		t.Fatal("unexpected alias")
	}
	if lookups != 2 {
		t.Fatalf("lookups = %d, want both supported shells", lookups)
	}
	if _, _, _, found := findShellAlias("not valid", lookPath, run); found {
		t.Fatal("invalid command name reported as an alias")
	}
	if lookups != 2 {
		t.Fatal("invalid command name should not inspect shells")
	}
}

// A preset launch must be deterministic: whatever the parent shell exported for
// either capacity variable, the dialect's own capacity is what Claude Code sees.
func TestClaudeEnvironmentOverridesInheritedContextWindow(t *testing.T) {
	dialect := presets["codex-sol"]
	dialect.Port = 43170
	dialect.APIKey = "local-secret"

	env := claudeEnvironment([]string{
		"PATH=/usr/bin", autoCompactWindowEnv + "=8000", maxContextTokensEnv + "=8000",
	}, "/tmp/claude", dialect)

	for _, key := range contextWindowEnvs {
		if count := countEnv(env, key); count != 1 {
			t.Fatalf("%s appears %d times, want exactly 1", key, count)
		}
		if value := lookupEnv(env, key); value != "372000" {
			t.Fatalf("%s = %q, want the codex-sol capacity %q", key, value, "372000")
		}
	}
}

// Every preset must hand Claude Code a denominator through both chains,
// otherwise the route it cannot recognize is left uncalibrated exactly as issue
// #44 describes, or reports its fill level against the 200,000-token default.
func TestClaudeEnvironmentSetsBothCapacityVariablesForEveryPreset(t *testing.T) {
	for _, name := range presetNames() {
		dialect := presets[name]
		dialect.Port = 43170
		dialect.APIKey = "local-secret"
		env := claudeEnvironment(nil, "/tmp/claude", dialect)
		for _, key := range contextWindowEnvs {
			if value := lookupEnv(env, key); value == "" {
				t.Errorf("preset %q launches without %s", name, key)
			}
		}
	}
}

// An unknown capacity cannot be improved on, so an ambient value the user
// exported themselves is preserved rather than dropped.
func TestClaudeEnvironmentKeepsAmbientWindowForUnknownCapacity(t *testing.T) {
	dialect := Dialect{Model: "custom-model", Port: 43170, APIKey: "local-secret"}

	env := claudeEnvironment([]string{
		autoCompactWindowEnv + "=250000", maxContextTokensEnv + "=250000",
	}, "/tmp/claude", dialect)

	for _, key := range contextWindowEnvs {
		if value := lookupEnv(env, key); value != "250000" {
			t.Fatalf("%s = %q, want the inherited %q", key, value, "250000")
		}
	}
}

// ExtraEnv is explicit per-dialect configuration, so it stays the last word for
// each variable independently.
func TestClaudeEnvironmentLetsExtraEnvOverrideEitherWindowVariable(t *testing.T) {
	for _, overridden := range contextWindowEnvs {
		dialect := presets["codex-sol"]
		dialect.Port = 43170
		dialect.APIKey = "local-secret"
		dialect.ExtraEnv = map[string]string{overridden: "123456"}

		env := claudeEnvironment(nil, "/tmp/claude", dialect)

		if value := lookupEnv(env, overridden); value != "123456" {
			t.Errorf("%s = %q, want the explicit extraEnv value %q", overridden, value, "123456")
		}
		if count := countEnv(env, overridden); count != 1 {
			t.Errorf("%s appears %d times, want exactly 1", overridden, count)
		}
		for _, other := range contextWindowEnvs {
			if other == overridden {
				continue
			}
			if value := lookupEnv(env, other); value != "372000" {
				t.Errorf("%s = %q, want the untouched codex-sol capacity %q", other, value, "372000")
			}
		}
	}
}

// The reproduced session sent 368,812 effective input tokens into a
// 372,000-token window with no compaction event. With the window declared,
// Claude Code has the denominator it needs to compact before exhaustion.
func TestCodexSolLaunchDeclaresTheWindowTheReproducedSessionLacked(t *testing.T) {
	const effectiveInput = 15020 + 353792

	dialect := presets["codex-sol"]
	dialect.Port = 43170
	dialect.APIKey = "local-secret"

	env := claudeEnvironment(nil, "/tmp/claude", dialect)
	for _, key := range contextWindowEnvs {
		if value := lookupEnv(env, key); value != "372000" {
			t.Fatalf("%s = %q, want %q", key, value, "372000")
		}
	}
	if percent := float64(effectiveInput) / float64(dialect.ContextWindow) * 100; percent < 99 || percent > 100 {
		t.Fatalf("reproduced fill = %.1f%%, want the reported 99.1%% of the configured window", percent)
	}
}

func TestClaudeEnvironmentKeepsExistingRoutingVariables(t *testing.T) {
	dialect := presets["kimi"]
	dialect.Port = 43171
	dialect.APIKey = "local-secret"

	env := claudeEnvironment(nil, "/tmp/claude-config", dialect)

	for key, want := range map[string]string{
		"CLAUDE_CONFIG_DIR":            "/tmp/claude-config",
		"ANTHROPIC_BASE_URL":           "http://127.0.0.1:43171",
		"ANTHROPIC_AUTH_TOKEN":         "local-secret",
		"ANTHROPIC_MODEL":              "kimi-k3",
		"ANTHROPIC_DEFAULT_OPUS_MODEL": "kimi-k3",
		"CLAUDE_CODE_SUBAGENT_MODEL":   "kimi-k3",
		autoCompactWindowEnv:           "262144",
		maxContextTokensEnv:            "262144",
	} {
		if value := lookupEnv(env, key); value != want {
			t.Errorf("%s = %q, want %q", key, value, want)
		}
	}
	for _, item := range env {
		if strings.HasPrefix(item, "ANTHROPIC_API_KEY=") {
			t.Error("launch environment must not define ANTHROPIC_API_KEY")
		}
	}
}
