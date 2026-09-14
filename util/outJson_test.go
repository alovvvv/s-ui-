package util

import (
	"encoding/json"
	"testing"

	"github.com/alireza0/s-ui/database/model"
)

// https://github.com/alireza0/s-ui/issues/1243
// Editing a naive inbound and clearing QUIC Congestion Control must remove
// the stale quic / quic_congestion_control keys from the stored out_json.
func TestFillOutJsonNaiveClearsStaleQuic(t *testing.T) {
	inbound := &model.Inbound{
		Type:    "naive",
		Tag:     "naive-in",
		Options: json.RawMessage(`{"listen_port": 443}`),
		// out_json left over from a previous save with QUIC enabled
		OutJson: json.RawMessage(`{"quic": true, "quic_congestion_control": "bbr"}`),
	}

	if err := FillOutJson(inbound, "example.com"); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(inbound.OutJson, &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out["quic"]; ok {
		t.Errorf("quic should be removed when quic_congestion_control is cleared, got %v", v)
	}
	if v, ok := out["quic_congestion_control"]; ok {
		t.Errorf("quic_congestion_control should be removed when cleared, got %v", v)
	}
}

func TestFillOutJsonNaiveSetsQuic(t *testing.T) {
	inbound := &model.Inbound{
		Type:    "naive",
		Tag:     "naive-in",
		Options: json.RawMessage(`{"listen_port": 443, "quic_congestion_control": "bbr_standard"}`),
		OutJson: json.RawMessage(`{}`),
	}

	if err := FillOutJson(inbound, "example.com"); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(inbound.OutJson, &out); err != nil {
		t.Fatal(err)
	}
	if out["quic"] != true {
		t.Errorf("quic = %v, want true", out["quic"])
	}
	if out["quic_congestion_control"] != "bbr" {
		t.Errorf("quic_congestion_control = %v, want bbr (mapped from bbr_standard)", out["quic_congestion_control"])
	}
}

// handshake_timeout is entered once on the server side, so out_json has to
// carry it over to the client config the same way it does the TLS versions.
// Client-only keys such as spoof stay untouched.
func TestFillOutJsonTlsHandshakeTimeout(t *testing.T) {
	inbound := &model.Inbound{
		Type:    "vless",
		Tag:     "vless-in",
		Options: json.RawMessage(`{"listen_port": 443}`),
		OutJson: json.RawMessage(`{}`),
		TlsId:   1,
		Tls: &model.Tls{
			Id:     1,
			Name:   "tls",
			Server: json.RawMessage(`{"enabled": true, "handshake_timeout": "20s"}`),
			Client: json.RawMessage(`{"spoof": "allowed.example.com", "spoof_method": "wrong-checksum"}`),
		},
	}

	if err := FillOutJson(inbound, "example.com"); err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(inbound.OutJson, &out); err != nil {
		t.Fatal(err)
	}
	tls, ok := out["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("tls is missing from out_json: %s", inbound.OutJson)
	}
	if tls["handshake_timeout"] != "20s" {
		t.Errorf("handshake_timeout should be copied to the client config, got %v", tls["handshake_timeout"])
	}
	if tls["spoof"] != "allowed.example.com" {
		t.Errorf("spoof should be kept, got %v", tls["spoof"])
	}
	if tls["spoof_method"] != "wrong-checksum" {
		t.Errorf("spoof_method should be kept, got %v", tls["spoof_method"])
	}
}
