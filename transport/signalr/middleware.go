package signalr

import (
	"net/http"
)

const (
	corsOptionMethod           string = "OPTIONS"
	corsAllowOriginHeader      string = "Access-Control-Allow-Origin"
	corsExposeHeadersHeader    string = "Access-Control-Expose-Headers"
	corsMaxAgeHeader           string = "Access-Control-Max-Age"
	corsAllowMethodsHeader     string = "Access-Control-Allow-Methods"
	corsAllowHeadersHeader     string = "Access-Control-Allow-Headers"
	corsAllowCredentialsHeader string = "Access-Control-Allow-Credentials"
)

func (s *Server) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//log.Info(r.RequestURI, r.Method)

		// 接受指定域的请求
		w.Header().Set(corsAllowOriginHeader, r.Header.Get("Origin"))
		// 设置服务器支持的所有跨域请求的方法
		w.Header().Set(corsAllowMethodsHeader, "POST,GET,OPTIONS,PUT,DELETE")

		// 服务器支持的所有头信息字段，不限于浏览器在"预检"中请求的字段
		w.Header().Set(corsAllowHeadersHeader, "Content-Type,x-requested-with,x-signalr-user-agent,Upgrade,Connection")
		w.Header().Set(corsAllowCredentialsHeader, "true")

		// 放行所有OPTIONS方法
		if r.Method == corsOptionMethod {
			w.WriteHeader(http.StatusOK)
			return
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}
