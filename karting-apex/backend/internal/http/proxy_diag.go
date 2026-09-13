package httpapi

import (
	"net"
	"net/http"
	"strings"
)

// ВРЕМЕННО: диагностика того, как прокси Render/Cloudflare передают IP клиента.
// Нужна один раз, чтобы выбрать неподделываемый источник IP для лимитов; удаляется
// следующим коммитом. Сами адреса не возвращаются — только структура: сколько
// адресов в цепочке, совпадает ли позиция с меткой из запроса (probe), приватный ли
// адрес. Сравнение с меткой показывает, какие позиции клиент может подделать.

type proxyDiagEntry struct {
	Index       int  `json:"index"`
	ValidIP     bool `json:"valid_ip"`
	EqualsProbe bool `json:"equals_probe"`
	Private     bool `json:"private"`
}

type proxyDiagHeader struct {
	Present        bool `json:"present"`
	ValidIP        bool `json:"valid_ip"`
	EqualsProbe    bool `json:"equals_probe"`
	Private        bool `json:"private"`
	EqualsXFFIndex int  `json:"equals_xff_index"`
}

type proxyDiagResponse struct {
	RemoteAddrPrivate bool                       `json:"remote_addr_private"`
	XFF               []proxyDiagEntry           `json:"xff"`
	Headers           map[string]proxyDiagHeader `json:"headers"`
}

func proxyDiagHandler(w http.ResponseWriter, r *http.Request) {
	probe := r.URL.Query().Get("probe")
	resp := proxyDiagResponse{Headers: map[string]proxyDiagHeader{}}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		resp.RemoteAddrPrivate = isPrivateIP(host)
	}

	var xff []string
	if raw := r.Header.Get("X-Forwarded-For"); raw != "" {
		for i, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			xff = append(xff, value)
			resp.XFF = append(resp.XFF, proxyDiagEntry{
				Index:       i,
				ValidIP:     net.ParseIP(value) != nil,
				EqualsProbe: probe != "" && value == probe,
				Private:     isPrivateIP(value),
			})
		}
	}

	for _, name := range []string{"CF-Connecting-IP", "True-Client-IP", "X-Real-IP", "X-Envoy-External-Address", "Forwarded", "CF-IPCountry"} {
		value := strings.TrimSpace(r.Header.Get(name))
		header := proxyDiagHeader{Present: value != "", EqualsXFFIndex: -1}
		if value != "" {
			header.ValidIP = net.ParseIP(value) != nil
			header.EqualsProbe = probe != "" && value == probe
			header.Private = isPrivateIP(value)
			for i, entry := range xff {
				if entry == value {
					header.EqualsXFFIndex = i
					break
				}
			}
		}
		resp.Headers[strings.ToLower(name)] = header
	}

	writeJSON(w, http.StatusOK, resp)
}

func isPrivateIP(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())
}
