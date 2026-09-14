package model

import (
	"encoding/json"
	"testing"
)

func TestEndpointMarshalKeepsInlineTLS(t *testing.T) {
	// OpenVPN and OpenConnect write their TLS options in their own field names,
	// so the panel stores them as the endpoint gave them and hands them over
	// untouched.
	inline := `{
		"server": "example.com",
		"server_port": 1194,
		"mode": "tls",
		"tls": {
			"certificate_path": "/etc/openvpn/ca.crt",
			"control_wrap": {"type": "tls_crypt", "key_path": "/etc/openvpn/tc.key"}
		}
	}`
	var endpoint Endpoint
	if err := endpoint.UnmarshalJSON([]byte(`{"type":"openvpn-client","tag":"ovpn",` + inline[1:])); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	encoded, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var tls struct {
		CertificatePath string `json:"certificate_path"`
		ControlWrap     struct {
			Type    string `json:"type"`
			KeyPath string `json:"key_path"`
		} `json:"control_wrap"`
	}
	if err = json.Unmarshal(got["tls"], &tls); err != nil {
		t.Fatalf("decode tls: %v", err)
	}
	if tls.CertificatePath != "/etc/openvpn/ca.crt" {
		t.Errorf("certificate_path = %q", tls.CertificatePath)
	}
	if tls.ControlWrap.Type != "tls_crypt" || tls.ControlWrap.KeyPath != "/etc/openvpn/tc.key" {
		t.Errorf("control_wrap = %+v", tls.ControlWrap)
	}
}

func TestEndpointUnmarshalDropsTlsId(t *testing.T) {
	// A panel built before endpoints carried their own TLS still sends tls_id.
	// Keeping it would put an unknown key in front of sing-box, which refuses
	// the whole config over it.
	var endpoint Endpoint
	err := endpoint.UnmarshalJSON([]byte(`{"type":"openconnect","tag":"oc","tls_id":3,"server":"vpn.example.com"}`))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	encoded, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, exists := got["tls_id"]; exists {
		t.Errorf("tls_id survived into the rendered config: %s", encoded)
	}
	if _, exists := got["tls"]; exists {
		t.Errorf("tls_id was turned into a tls object: %s", encoded)
	}
}

func TestEndpointMarshalWarpRendersAsWireguard(t *testing.T) {
	var endpoint Endpoint
	if err := endpoint.UnmarshalJSON([]byte(`{"type":"warp","tag":"warp-1","mtu":1420}`)); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	encoded, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		Type string `json:"type"`
	}
	if err = json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Type != "wireguard" {
		t.Errorf("type = %q, want wireguard", got.Type)
	}
}
