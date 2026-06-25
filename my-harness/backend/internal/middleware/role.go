package middleware

import (
	"net/http"
)

// RequireRole 校验当前用户角色，不符合则返回 403。
// role 由 CognitoJWTAuth 中间件注入 roleKey context。
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(roleKey).(string)
			if !ok || role == "" {
				writeErrorJSON(w, http.StatusForbidden, "forbidden")
				return
			}
			if _, permitted := allowed[role]; !permitted {
				writeErrorJSON(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
