package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/alireza0/s-ui/database"

	"gorm.io/gorm"
)

func settingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if err := database.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatal(err)
	}
	return database.GetDB()
}

// The settings form was write-anything. GetAllSetting hid these four, but Save
// took whatever keys it was posted: a client could rewrite the session secret,
// logging every session out, or replace the whole sing-box base config through
// a form that has no business touching it.
func TestSaveRejectsProtectedSettings(t *testing.T) {
	db := settingTestDB(t)
	s := &SettingService{}

	for key := range protectedSettings {
		t.Run(key, func(t *testing.T) {
			before, err := s.getString(key)
			if err != nil {
				t.Fatalf("reading %s: %v", key, err)
			}

			payload, _ := json.Marshal(map[string]string{key: "taken-over"})
			if err := s.Save(db, payload); err != nil {
				t.Fatalf("Save returned an error for %s: %v", key, err)
			}

			after, err := s.getString(key)
			if err != nil {
				t.Fatalf("re-reading %s: %v", key, err)
			}
			if after != before {
				t.Errorf("%s was overwritten through the settings endpoint: %q -> %q", key, before, after)
			}
		})
	}
}

// And they are not handed out either, so the frontend never has them to post
// back in the first place.
func TestGetAllSettingWithholdsProtectedSettings(t *testing.T) {
	settingTestDB(t)
	s := &SettingService{}

	all, err := s.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	for key := range protectedSettings {
		if _, present := (*all)[key]; present {
			t.Errorf("%s was returned to the client", key)
		}
	}
}

// Ordinary settings still save, or the allowlist would have broken the form it
// is protecting.
func TestSaveStillWritesOrdinarySettings(t *testing.T) {
	db := settingTestDB(t)
	s := &SettingService{}

	payload, _ := json.Marshal(map[string]string{"subUpdates": "24", "timeLocation": "UTC"})
	if err := s.Save(db, payload); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, _ := s.getString("subUpdates"); got != "24" {
		t.Errorf("subUpdates = %q, want 24", got)
	}
	if got, _ := s.getString("timeLocation"); got != "UTC" {
		t.Errorf("timeLocation = %q, want UTC", got)
	}
}

// An unknown key used to UPDATE zero rows and report success.
func TestSaveRejectsUnknownSettings(t *testing.T) {
	db := settingTestDB(t)
	s := &SettingService{}

	payload, _ := json.Marshal(map[string]string{"webPrt": "2096"})
	if err := s.Save(db, payload); err == nil {
		t.Error("a misspelled setting key was accepted silently")
	}
}
