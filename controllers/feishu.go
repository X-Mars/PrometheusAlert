package controllers

import (
	"PrometheusAlert/models"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type FSMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func PostToFS(title, text, Fsurl, userEmail, logsign string) string {
	open := beego.AppConfig.String("open-feishu")
	if open != "1" {
		logs.Info(logsign, "[feishu]", "飞书接口未配置未开启状态,请先配置open-feishu为1")
		return "飞书接口未配置未开启状态,请先配置open-feishu为1"
	}
	RTstring := ""
	if strings.Contains(Fsurl, "/v2/") {
		RTstring = PostToFeiShuv2(title, text, Fsurl, userEmail, logsign)
	} else {
		RTstring = PostToFeiShu(title, text, Fsurl, logsign)
	}
	return RTstring
}

func PostToFeiShu(title, text, Fsurl, logsign string) string {
	u := FSMessage{Title: title, Text: text}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(u)
	logs.Info(logsign, "[feishu]", b)
	var tr *http.Transport
	if proxyUrl := beego.AppConfig.String("proxy"); proxyUrl != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyUrl)
		}
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           proxy,
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	client := &http.Client{Transport: tr}
	res, err := client.Post(Fsurl, "application/json", b)
	if err != nil {
		logs.Error(logsign, "[feishu]", err.Error())
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logs.Error(logsign, "[feishu]", err.Error())
	}
	models.AlertToCounter.WithLabelValues("feishu").Add(1)
	ChartsJson.Feishu += 1
	logs.Info(logsign, "[feishu]", string(result))
	return string(result)
}

type Conf struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

type Te struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type Element struct {
	Tag      string    `json:"tag"`
	Text     Te        `json:"text"`
	Content  string    `json:"content"`
	Elements []Element `json:"elements"`
}

type Titles struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

type Headers struct {
	Title    Titles `json:"title"`
	Template string `json:"template"`
}

type Cards struct {
	Config   Conf      `json:"config"`
	Elements []Element `json:"elements"`
	Header   Headers   `json:"header"`
}

type FSMessagev2 struct {
	MsgType string `json:"msg_type"`
	Email   string `json:"email"` //@所使用字段
	Card    Cards  `json:"card"`
}

type TenantAccessMeg struct {
	AppId     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

type TenantAccessResp struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
}

// 从text中取出告警名称、告警状态，当告警状态为resolved时，返回 告警信息 + 告警名称，其他值时，返回 恢复信息 + 告警名称 text例子如下
///PrometheusAlert

// Prometheus 恢复信息

// 告警名称： ContainerRestart
// 告警级别： warning
// 告警状态： resolved
// 开始时间： 2025-05-08T04:36:27.678Z
// 结束时间： 2025-05-08T05:13:42.678Z
// 告警主机： dev-2

// 容器 doc-sync-api 发生重启，请及时查看!

func GetAlertName(text string) string {
	// 解析告警名称
	lines := strings.Split(text, "\n")
	alertname := ""
	status := ""
	instance := ""
	for _, line := range lines {
	 if strings.Contains(line, "告警名称：") {
		parts := strings.Split(line, "：")
		if len(parts) > 1 {
		 alertname = strings.TrimSpace(parts[1])
		}
	 } else if strings.Contains(line, "告警状态：") {
				parts := strings.Split(line, "：")
				if len(parts) > 1 {
				alertstatus := strings.TrimSpace(parts[1])
				if alertstatus == "resolved" {
					// 解析告警名称
					status = "恢复"
				} else {
					// 解析告警名称
					status = "告警"
				}
			}
		} else if strings.Contains(line, "告警主机：") {
			parts := strings.Split(line, "：")
			if len(parts) > 1 {
				instance = strings.TrimSpace(parts[1])
			}
		}
	}
	return "Prometheus " + status + " - " + instance + " " + alertname
 }


func PostToFeiShuv2(title, text, Fsurl, userOpenId, logsign string) string {
	title = GetAlertName(text)
	var color string
	if strings.Count(text, "resolved") > 0 && strings.Count(text, "firing") > 0 {
		color = "orange"
	} else if strings.Count(text, "resolved") > 0 {
		color = "green"
	} else {
		color = "red"
	}

	SendContent := text
	if userOpenId != "" {
		OpenIds := strings.Split(userOpenId, ",")
		OpenIdtext := ""
		for _, OpenId := range OpenIds {
			OpenIdtext += "<at user_id=" + OpenId + " id=" + OpenId + " email=" + OpenId  + "></at>"
		}
		SendContent += OpenIdtext
	}

	u := FSMessagev2{
		MsgType: "interactive",
		Email:   "xxxxxxxxxxx@qq.com",
		Card: Cards{
			Config: Conf{
				WideScreenMode: true,
				EnableForward:  true,
			},
			Header: Headers{
				Title: Titles{
					Content: title,
					Tag:     "plain_text",
				},
				Template: color,
			},
			Elements: []Element{
				Element{
					Tag: "div",
					Text: Te{
						Content: SendContent,
						Tag:     "lark_md",
					},
				},
				{
					Tag: "hr",
				},
				{
					Tag: "note",
					Elements: []Element{
						{
							Content: title,
							Tag:     "lark_md",
						},
					},
				},
			},
		},
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(u)
	logs.Info(logsign, "[feishuv2]", b)
	var tr *http.Transport
	if proxyUrl := beego.AppConfig.String("proxy"); proxyUrl != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyUrl)
		}
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           proxy,
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	client := &http.Client{Transport: tr}
	res, err := client.Post(Fsurl, "application/json", b)
	if err != nil {
		logs.Error(logsign, "[feishuv2]", title+": "+err.Error())
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logs.Error(logsign, "[feishuv2]", title+": "+err.Error())
	}
	models.AlertToCounter.WithLabelValues("feishu").Add(1)
	ChartsJson.Feishu += 1
	logs.Info(logsign, "[feishuv2]", title+": "+string(result))
	return string(result)
}
