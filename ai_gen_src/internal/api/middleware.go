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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-OpsOne-Role")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// actorFromRequest returns user id for audit (§2.6 dev bypass).
func actorFromRequest(r *http.Request, bypass bool) string {
	if v := r.Header.Get("X-OpsOne-Actor"); v != "" {
		return v
	}
	if bypass {
		return "dev.admin"
	}
	return "unknown"
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
