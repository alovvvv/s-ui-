package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/alireza0/s-ui/database"
)

// Maintenance mode has to survive every path that starts the core. These tests
// call the guards with no core attached at all: reaching the core would panic,
// so passing proves nothing downstream of the guard runs.
func maintenanceService(t *testing.T, enabled bool) *ConfigService {
	t.Helper()
	if err := database.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	s := &ConfigService{}
	if err := s.SettingService.SetMaintenance(enabled); err != nil {
		t.Fatal(err)
	}
	stored, err := s.SettingService.GetMaintenance()
	if err != nil {
		t.Fatal(err)
	}
	if stored != enabled {
		t.Fatalf("maintenance stored as %v, want %v", stored, enabled)
	}
	return s
}

// The five-second watchdog exists to restart a crashed core, and goes through
// StartCore to do it. Without this guard it would undo maintenance mode within
// five seconds of it being turned on.
func TestStartCoreStopsAtMaintenance(t *testing.T) {
	s := maintenanceService(t, true)
	if err := s.StartCore(); err != nil {
		t.Fatalf("StartCore: %v", err)
	}
}

func TestRestartCoreRefusedInMaintenance(t *testing.T) {
	s := maintenanceService(t, true)
	err := s.RestartCore()
	if err == nil {
		t.Fatal("RestartCore succeeded while stopped for maintenance")
	}
	if !strings.Contains(err.Error(), "maintenance") {
		t.Errorf("unexpected error: %v", err)
	}
}

// A base config change restarts the core with the new config. In maintenance
// the config is still saved; it just does not take effect yet.
func TestRestartCoreWithConfigStopsAtMaintenance(t *testing.T) {
	s := maintenanceService(t, true)
	if err := s.restartCoreWithConfig([]byte(`{"log":{"level":"info"}}`)); err != nil {
		t.Fatalf("restartCoreWithConfig: %v", err)
	}
}

// An unreadable setting must not leave the core stopped: a panel that cannot
// read its own settings should still come up serving clients.
func TestMaintenanceDefaultsOffWhenUnset(t *testing.T) {
	if err := database.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	s := &ConfigService{}
	if s.inMaintenance() {
		t.Error("maintenance reads as on before it was ever set")
	}
}

// The settings page posts every key it was given back again, maintenance among
// them. Writing it from there would set the flag without stopping the core, so
// SettingService.Save has to leave it to the action that does both.
func TestSettingsSaveIgnoresMaintenance(t *testing.T) {
	if err := database.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	s := &SettingService{}
	// Save updates existing rows, and the defaults are only written on first
	// read, which is what the settings page does before it ever saves.
	if _, err := s.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMaintenance(true); err != nil {
		t.Fatal(err)
	}

	err := s.Save(database.GetDB(), []byte(`{"maintenance":"false","timeLocation":"Europe/Berlin"}`))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	maintenance, err := s.GetMaintenance()
	if err != nil {
		t.Fatal(err)
	}
	if !maintenance {
		t.Error("the settings form turned maintenance off without starting the core")
	}
	// Everything else in the same payload still has to be written.
	location, err := s.getString("timeLocation")
	if err != nil {
		t.Fatal(err)
	}
	if location != "Europe/Berlin" {
		t.Errorf("timeLocation = %q, want Europe/Berlin", location)
	}
}
