package util

func ShadowsocksClientConfigKey(method string) string {
	switch method {
	case "2022-blake3-aes-128-gcm":
		return "shadowsocks16"
	default:
		return "shadowsocks"
	}
}
