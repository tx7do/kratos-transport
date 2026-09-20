package signalr

import (
	"net/http"
	"strings"
)

const (
	corsOptionMethod           string = "OPTIONS"
	corsAllowOriginHeader      string = "Access-Control-Allow-Origin"
	corsVaryHeader             string = "Vary"
	corsExposeHeadersHeader    string = "Access-Control-Expose-Headers"
	corsMaxAgeHeader           string = "Access-Control-Max-Age"
	corsAllowMethodsHeader     string = "Access-Control-Allow-Methods"
	corsAllowHeadersHeader     string = "Access-Control-Allow-Headers"
	corsAllowCredentialsHeader string = "Access-Control-Allow-Credentials"
)

// CORS 跨域中间件。
// 安全语义：默认不再反射任意 Origin（旧实现“反射 Origin + Allow-Credentials”
// 等价于向任意站点开放凭据型跨域）。仅当通过 WithAllowedOrigins 配置了白名单，
// 且请求 Origin 命中白名单时才放行；通配符 "*" 用于公开接口（不带凭据）。
func (s *Server) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := s.allowedOrigin(origin)

		if r.Method == corsOptionMethod {
			if allowed {
				w.Header().Set(corsAllowOriginHeader, origin)
				w.Header().Set(corsVaryHeader, "Origin")
				w.Header().Set(corsAllowMethodsHeader, "POST,GET,OPTIONS,PUT,DELETE")
				w.Header().Set(corsAllowHeadersHeader, "Content-Type,x-requested-with,x-signalr-user-agent,Upgrade,Connection")
				w.Header().Set(corsMaxAgeHeader, "86400")
				if s.allowCredentials {
					w.Header().Set(corsAllowCredentialsHeader, "true")
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if allowed {
			w.Header().Set(corsAllowOriginHeader, origin)
			w.Header().Set(corsVaryHeader, "Origin")
			w.Header().Set(corsExposeHeadersHeader, corsAllowOriginHeader)
			if s.allowCredentials {
				w.Header().Set(corsAllowCredentialsHeader, "true")
			}
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

// allowedOrigin 判断 origin 是否在白名单内。
// 未配置白名单时默认同源可用（无 CORS 头）；"*" 表示全部放行。
func (s *Server) allowedOrigin(origin string) bool {
	if len(s.allowedOrigins) == 0 {
		return false
	}
	for _, o := range s.allowedOrigins {
		if o == "*" {
			// 通配符与凭据不能组合使用（等价于向任意站点开放凭据型跨域）
			return !s.allowCredentials
		}
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}
