package dns

import (
"fmt"
"github.com/jeessy2/ddns-go/v6/util"
"log"
"net/http"
"net/url"
"strings"
)

// Dynu 信息
type Dynu struct {
Username string
Password string
Domain   string
SubDomain string
}

// 通用返回信息
type DynuResult struct {
Success bool   `json:"success"`
Message string `json:"message"`
}

// 域名解析记录
type DynuRecord struct {
Id   int    `json:"id"`
Name string `json:"name"`
}

// 查询域名解析记录返回信息
type DynuQueryResult struct {
Success bool        `json:"success"`
Message string      `json:"message"`
Array   []DynuRecord `json:"array"`
}

// 获取域名解析记录
func (dynu *Dynu) GetDnsRecord(ipType int) (string, string) {
// 获取 Token
token, err := dynu.getToken()
if err != nil {
return "", ""
}

// 查询记录
subDomain := ""
if dynu.SubDomain != "" {
subDomain = dynu.SubDomain + "."
}
recordUrl := fmt.Sprintf("https://api.dynu.com/v2/dns/%s/%s", token, url.PathEscape(dynu.Domain))
respBody := util.HttpGetWithHeaders(recordUrl, map[string]string{
"User-Agent": util.UserAgent,
})
var queryResult DynuQueryResult
util.JSON.Unmarshal(respBody, &queryResult)

if !queryResult.Success {
log.Printf("Query Dynu record failed: %s", queryResult.Message)
return "", ""
}

var recordId int
for _, record := range queryResult.Array {
if record.Name == subDomain+dynu.Domain {
recordId = record.Id
break
}
}
if recordId == 0 {
log.Printf("Domain %s not found in Dynu", subDomain+dynu.Domain)
return "", ""
}

return fmt.Sprintf("%d", recordId), ""
}

// 新增域名解析记录
func (dynu *Dynu) AddDnsRecord(ipType int, domainId string, value string) bool {
// 获取 Token
token, err := dynu.getToken()
if err != nil {
return false
}

// 构建更新 URL
// Dynu API v2 使用 PUT 方法更新记录
updateUrl := fmt.Sprintf("https://api.dynu.com/v2/dns/%s/%s/%s", token, url.PathEscape(dynu.Domain), domainId)
// 注意：Dynu v2 API 需要发送 JSON 数据包
jsonBody := fmt.Sprintf(`{"content":"%s"}`, value)

// 使用 PUT 请求发送更新
req, err := http.NewRequest("PUT", updateUrl, strings.NewReader(jsonBody))
if err != nil {
log.Printf("Create request failed: %v", err)
return false
}
req.Header.Set("User-Agent", util.UserAgent)
req.Header.Set("Content-Type", "application/json")

client := &http.Client{}
resp, err := client.Do(req)
if err != nil {
log.Printf("Update Dynu record failed: %v", err)
return false
}
defer resp.Body.Close()

body := util.Reader2Bytes(resp.Body)
var result DynuResult
util.JSON.Unmarshal(body, &result)

if result.Success {
log.Printf("Update Dynu record success: %s", value)
return true
} else {
log.Printf("Update Dynu record failed: %s", result.Message)
return false
}
}

// 获取认证 Token
// Dynu API v2 需要先通过 Basic Auth 获取 Token
func (dynu *Dynu) getToken() (string, error) {
authUrl := "https://api.dynu.com/v2/auth"
req, err := http.NewRequest("POST", authUrl, nil)
if err != nil {
log.Printf("Create auth request failed: %v", err)
return "", err
}
req.SetBasicAuth(dynu.Username, dynu.Password)
req.Header.Set("User-Agent", util.UserAgent)

client := &http.Client{}
resp, err := client.Do(req)
if err != nil {
log.Printf("Auth to Dynu failed: %v", err)
return "", err
}
defer resp.Body.Close()

body := util.Reader2Bytes(resp.Body)
// Dynu Auth 返回示例: {"id":12345,"token":"abc-def-ghi"}
var authResult map[string]interface{}
util.JSON.Unmarshal(body, &authResult)

if token, ok := authResult["token"].(string); ok {
return token, nil
} else {
log.Printf("Get Dynu token failed: Invalid response %s", string(body))
return "", fmt.Errorf("get token failed")
}
}