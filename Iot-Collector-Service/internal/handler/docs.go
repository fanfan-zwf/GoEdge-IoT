/*
* 日期: 2026.11.13
* 作者: 范范zwf
* 作用: 接口清单页 —— 浏览器打开 API 根路径即可查看全部已注册接口
*
* 清单完全来自 gin 自带的 Engine.Routes()（遍历路由树得到 Method/Path/Handler），
* 本文件不含任何手写接口表：注册了什么就显示什么，永远不会与代码脱节，也无需维护。
 */
package handler

import (
	"html"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 清单页路径
const (
	PathDocs      = "/api/v1.0/docs" // ?format=json 可拿到机器可读清单
	PathDocsRoot  = "/api/v1.0/"     // API 根路径，同样返回清单页
	PathDocsSlash = "/"              // 站点根，302 跳到清单页
)

// docs_api_register 注册接口清单页。清单在请求时才取 r.Routes()，故与其它 register 的先后无关。
func docs_api_register(r *gin.Engine) {
	index := func(ctx *gin.Context) {
		routes := docsSortedRoutes(r.Routes())

		if strings.EqualFold(ctx.Query("format"), "json") {
			ctx.Set("Response", []any{200, "ok", gin.H{
				"total":     len(routes),
				"generated": time.Now().Format(time.RFC3339Nano),
				"items":     routes,
			}})
			return
		}

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", docsHTML(routes, ctx.Request.Host))
	}

	r.GET(PathDocs, index)
	r.GET(PathDocsRoot, index)
	r.GET(PathDocsSlash, func(ctx *gin.Context) { ctx.Redirect(http.StatusFound, PathDocs) })
}

// docsRoute 清单页/JSON 上的一条接口
type docsRoute struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"` // 处理函数名，便于按名字定位代码
	Auth    string `json:"auth"`
}

// docsHandlerName 把 runtime 函数全名简化成可读形式：
// collector/internal/handler.api_drive_config_query -> api_drive_config_query
// collector/internal/handler.docs_api_register.func1 -> docs_api_register.func1（注册函数里的闭包）
func docsHandlerName(full string) string {
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:] // 去掉包路径，保留 handler. 前缀供下一步剔除
	}
	return strings.TrimPrefix(full, "handler.")
}

// docsSortedRoutes 取 gin 内置路由表，简化处理函数名并补上凭证要求，按路径排序
func docsSortedRoutes(routes gin.RoutesInfo) []docsRoute {
	list := make([]docsRoute, 0, len(routes))
	for _, rt := range routes {
		list = append(list, docsRoute{
			Method:  rt.Method,
			Path:    rt.Path,
			Handler: docsHandlerName(rt.Handler),
			Auth:    docsAuth(rt.Path),
		})
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Path != list[j].Path {
			return list[i].Path < list[j].Path
		}
		return list[i].Method < list[j].Method
	})
	return list
}

// docsAuth 凭证要求直接问中间件的豁免判断，不另维护一份名单
func docsAuth(path string) string {
	switch path {
	case PathPointWS:
		return "Bearer（子协议位）"
	case PathDocs, PathDocsRoot, PathDocsSlash:
		return "免鉴权"
	}
	if authExemptPath(path) {
		return "免鉴权"
	}
	return "Bearer / HMAC"
}

// docsHTML 极简静态页：标题 + 过滤框 + 一张表
func docsHTML(list []docsRoute, host string) []byte {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>接口清单</title><style>` + docsCSS + `</style></head><body>` +
		`<div class="wrap"><h1>接口清单</h1><p class="sub">` + html.EscapeString(host) + ` · ` +
		`<span id="n"></span> 条（由 gin 运行时路由表生成）</p>` +
		`<input id="q" placeholder="过滤路径 / 方法 / 处理函数" autofocus>` +
		`<table><thead><tr><th>方法</th><th>路径</th><th>处理函数</th><th>凭证</th></tr></thead><tbody>`)

	for _, it := range list {
		pathCell := `<code>` + html.EscapeString(it.Path) + `</code>`
		if it.Method == http.MethodGet && it.Path != PathDocsSlash {
			pathCell = `<a href="` + html.EscapeString(it.Path) + `" target="_blank">` + pathCell + `</a>`
		}
		b.WriteString(`<tr><td class="m ` + it.Method + `">` + html.EscapeString(it.Method) + `</td><td>` +
			pathCell + `</td><td class="h">` + html.EscapeString(it.Handler) + `</td><td class="a">` +
			html.EscapeString(it.Auth) + `</td></tr>`)
	}

	b.WriteString(`</tbody></table><p class="foot">清单页免鉴权，不含业务数据。机器可读：` +
		PathDocs + `?format=json</p></div><script>` + docsJS + `</script></body></html>`)
	return []byte(b.String())
}

const docsCSS = `
:root{--bg:#0f141a;--card:#161d26;--line:#243040;--fg:#dce6f2;--dim:#8b9bb0;--amber:#e8a33d;--get:#3fa7d6;--post:#5bbf7a}
@media(prefers-color-scheme:light){:root{--bg:#f5f7fa;--card:#fff;--line:#dfe5ec;--fg:#1f2937;--dim:#6b7a8d}}
*{box-sizing:border-box}
body{margin:0;padding:26px 16px 50px;background:var(--bg);color:var(--fg);
font:14px/1.6 ui-monospace,SFMono-Regular,Menlo,Consolas,"Noto Sans Mono CJK SC",monospace}
.wrap{max-width:1100px;margin:0 auto}
h1{font-size:19px;margin:0 0 4px}
.sub{color:var(--dim);margin:0 0 14px}
input{width:100%;padding:9px 12px;margin-bottom:14px;background:var(--card);border:1px solid var(--line);
border-radius:8px;color:var(--fg);font:inherit}
input:focus{outline:2px solid var(--amber);outline-offset:-1px}
table{width:100%;border-collapse:collapse;background:var(--card);border:1px solid var(--line);border-radius:8px;overflow:hidden}
th{text-align:left;padding:9px 12px;color:var(--dim);font-weight:400;font-size:12px;border-bottom:1px solid var(--line)}
td{padding:7px 12px;border-bottom:1px solid var(--line);vertical-align:top}
.m{width:56px;font-weight:700}
.GET{color:var(--get)}
.POST{color:var(--post)}
.h{width:280px;color:var(--dim)}
.a{width:140px;color:var(--dim);font-size:12px}
code{font-size:12.5px;word-break:break-all}
a{text-decoration:none}
a:hover code{color:var(--amber);text-decoration:underline}
.foot{color:var(--dim);font-size:12px;margin-top:16px}
tr.hide{display:none}
`

const docsJS = `
var q=document.getElementById('q'),rows=[].slice.call(document.querySelectorAll('tbody tr'));
function filter(){
  var k=(q.value||'').trim().toLowerCase(),n=0;
  rows.forEach(function(tr){
    var hit=!k||tr.textContent.toLowerCase().indexOf(k)>=0;
    tr.classList.toggle('hide',!hit);
    if(hit)n++;
  });
  document.getElementById('n').textContent=n;
}
q.addEventListener('input',filter);
filter();
`
