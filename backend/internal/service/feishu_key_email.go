package service

import (
	"bytes"
	"html/template"
	"strings"
)

var feishuKeyEmailTemplate = template.Must(template.New("feishu-key").Parse(`<!doctype html><html lang="zh-CN"><body style="margin:0;padding:24px;background:#f4f5fa;color:#1f2937;font-family:Arial,sans-serif"><main style="max-width:640px;margin:auto;background:white;padding:28px;border-radius:12px"><h2>{{.Name}}，您好</h2><p style="white-space:pre-wrap">{{.Introduction}}</p><h3>API 地址</h3><p><a href="{{.APIURL}}">{{.APIURL}}</a></p><h3>您的专属 API Key</h3><p style="padding:16px;background:#f3f4f6;word-break:break-all;font-family:monospace">{{.Key}}</p><p>来源分组：{{.Groups}}</p><p>账户余额：${{printf "%.8f" .Balance}}<br>Key 总消费上限：{{if gt .Quota 0.0}}${{printf "%.8f" .Quota}}{{else}}不限额{{end}}<br>有效期：{{.Expiry}}</p><p style="color:#6b7280;font-size:13px">Key 额度是消费上限，实际可用额度取决于账户余额、订阅及站点限制。此 Key 仅属于您本人，请勿转发。重复补发会复用同一把 Key，不会重复赠额。</p></main></body></html>`))

func renderFeishuKeyEmail(cfg FeishuKeyDeliveryConfig, row FeishuKeyDeliveryRow, key string) (string, error) {
	expiry := "长期有效"
	if row.ExpiresAt != nil {
		expiry = row.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC")
	}
	data := struct {
		Name, Introduction, APIURL, Key, Groups, Expiry string
		Balance, Quota                                  float64
	}{row.Name, cfg.Introduction, cfg.APIURL, key, strings.Join(row.GroupNames, "、"), expiry, row.Balance, row.Quota}
	var out bytes.Buffer
	err := feishuKeyEmailTemplate.Execute(&out, data)
	return out.String(), err
}
