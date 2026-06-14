package api

import (
	"net/http"
	"strings"
)

// withCORS adds CORS headers per §2.1.
func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-OpsOne-Role, X-OpsOne-Actor")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// actorFromRequest returns user id for audit (§2.6 dev bypass).
// The Fetch API encodes header values as Latin-1, so non-ASCII names like
// "Khiêm" arrive as Latin-1 bytes. We reinterpret each byte as its Unicode
// codepoint (Latin-1 ↔ Unicode is a 1-to-1 identity mapping) to recover
// the correct UTF-8 string before writing to MySQL.
func actorFromRequest(r *http.Request, bypass bool) string {
	if v := r.Header.Get("X-OpsOne-Actor"); v != "" {
		return latin1HeaderToUTF8(v)
	}
	if bypass {
		return "dev.admin"
	}
	return "unknown"
}

// latin1HeaderToUTF8 converts a string whose bytes are Latin-1 encoded into
// a proper UTF-8 Go string. If the input is already valid UTF-8 (ASCII-only
// names), it is returned unchanged.
func latin1HeaderToUTF8(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			// Has non-ASCII Latin-1 bytes — convert each byte to its Unicode rune.
			runes := make([]rune, len(s))
			for j, b := range []byte(s) {
				runes[j] = rune(b)
			}
			return string(runes)
		}
	}
	return s // pure ASCII — no conversion needed
}

// roleFromRequest returns Admin or Ops.
func roleFromRequest(r *http.Request, bypass bool) string {
	if bypass {
		if role := r.Header.Get("X-OpsOne-Role"); role != "" {
			return strings.ToLower(role)
		}
		return "Admin"
	}
	return r.Header.Get("X-OpsOne-Role")
}

func requireAdmin(w http.ResponseWriter, r *http.Request, bypass bool) bool {
	role := strings.ToLower(roleFromRequest(r, bypass))
	if role == "admin" || role == "administrator" {
		return true
	}
	writeError(w, http.StatusForbidden, "forbidden", "Cần quyền Admin")
	return false
}

func requireAdminSilent(r *http.Request, bypass bool) bool {
	role := strings.ToLower(roleFromRequest(r, bypass))
	return role == "admin" || role == "administrator"
}
