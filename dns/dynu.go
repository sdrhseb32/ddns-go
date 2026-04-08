package dns

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sdrhseb32/ddns-go/config"
	"github.com/sdrhseb32/ddns-go/util"
)

// Dynu 提供商结构体
type Dynu struct {
	DNSServers []string
	config     *config.DNSConfig
}

// dynu域名信息响应结构体
type dynuDomainInfo struct {
	ID     int    `json:"id"`
	Domain string `json:"domain"`
	IPv4   string `json:"ipv4Address"`
	IPv6   string `json:"ipv6Address"`
}

// dynu更新IP请求结构体
type dynuUpdateRequest struct {
	IPv4 string `json:"ipv4Address,omitempty"`
	IPv6 string `json:"ipv6Address,omitempty"`
}

// 初始化Dynu
func NewDynu(dnsConfig *config.DNSConfig) *Dynu {
	return &Dynu{
		config: dnsConfig,
	}
}

// 初始化
func (d *Dynu) Init() error {
	// Dynu 无需额外初始化
	return nil
}

// 新增或更新域名解析
func (d *Dynu) AddUpdateDomainRecords() (domains []string, err error) {
	// 获取所有域名
	domainList := util.GetDomainList(d.config.Domain)
	if len(domainList) == 0 {
		return domains, errors.New("域名不能为空")
	}

	// 遍历所有域名更新
	for _, domain := range domainList {
		// 1. 获取域名ID
		domainID, err := d.getDomainID(domain)
		if err != nil {
			return domains, fmt.Errorf("获取域名ID失败: %w", err)
		}

		// 2. 更新IP
		err = d.updateIP(domainID)
		if err != nil {
			return domains, fmt.Errorf("更新域名失败: %w", err)
		}

		domains = append(domains, domain)
	}

	return domains, nil
}

// getDomainID 通过API获取域名ID
func (d *Dynu) getDomainID(domain string) (int, error) {
	client := util.GetHTTPClient()
	req, err := http.NewRequest("GET", "https://api.dynu.com/v2/dns", nil)
	if err != nil {
		return 0, err
	}

	// 设置请求头
	d.setAuthHeader(req)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		Domains []dynuDomainInfo `json:"domains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// 匹配域名
	for _, d := range result.Domains {
		if strings.EqualFold(d.Domain, domain) {
			return d.ID, nil
		}
	}

	return 0, fmt.Errorf("未找到域名: %s", domain)
}

// updateIP 更新IP地址
func (d *Dynu) updateIP(domainID int) error {
	// 构建更新请求
	updateReq := dynuUpdateRequest{}
	if d.config.IPv4 != "" {
		updateReq.IPv4 = d.config.IPv4
	}
	if d.config.IPv6 != "" {
		updateReq.IPv6 = d.config.IPv6
	}

	// 转换为JSON
	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	// 创建请求
	url := fmt.Sprintf("https://api.dynu.com/v2/dns/%d", domainID)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	// 设置请求头
	d.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := util.GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API请求失败, 状态码: %d", resp.StatusCode)
	}

	return nil
}

// setAuthHeader 设置认证头
func (d *Dynu) setAuthHeader(req *http.Request) {
	// Dynu API 使用 API Key 认证
	req.Header.Set("API-Key", d.config.Secret)
}
