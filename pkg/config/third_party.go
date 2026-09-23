package config

import "strings"

// ThirdPartyConfig 微信 / 支付宝 / 华为开放平台与商户凭证（仅服务端）。
// 客户端 AppID/UL 在 Flutter WysWechatConfig；此处 AppSecret、商户密钥、支付宝私钥、华为 Client Secret 不可下发 App。
type ThirdPartyConfig struct {
	WeChatAppID         string `mapstructure:"wechat_app_id"`
	WeChatAppSecret     string `mapstructure:"wechat_app_secret"`
	WeChatMchID         string `mapstructure:"wechat_mch_id"`
	WeChatMchAPIKey     string `mapstructure:"wechat_mch_api_key"`
	WeChatMchNotifyURL  string `mapstructure:"wechat_mch_notify_url"`
	AlipayAppID         string `mapstructure:"alipay_app_id"`
	AlipayPrivateKey    string `mapstructure:"alipay_private_key"`
	AlipayNotifyURL     string `mapstructure:"alipay_notify_url"`
	HuaweiClientID     string          `mapstructure:"huawei_client_id"`
	HuaweiClientSecret string          `mapstructure:"huawei_client_secret"`
	HuaweiRedirectURI  string          `mapstructure:"huawei_redirect_uri"`
	HuaweiIAP          HuaweiIAPConfig `mapstructure:"huawei_iap"`
	AppleIAP           AppleIAPConfig  `mapstructure:"apple_iap"`
	JPushAppKey         string `mapstructure:"jpush_app_key"`
	JPushMasterSecret   string `mapstructure:"jpush_master_secret"`
	JPushAPNsProduction bool   `mapstructure:"jpush_apns_production"`
}

// HuaweiIAPConfig AGC 应用内支付服务端密钥（JWT）。
type HuaweiIAPConfig struct {
	AppID               string `mapstructure:"app_id"`
	IssuerID            string `mapstructure:"issuer_id"`
	KeyID               string `mapstructure:"key_id"`
	PrivateKeyPEM       string `mapstructure:"private_key_pem"`
	SubscriptionRootURL string `mapstructure:"subscription_root_url"`
}

// Configured 是否具备发 JWT 所需字段。
func (h HuaweiIAPConfig) Configured() bool {
	return strings.TrimSpace(h.AppID) != "" &&
		strings.TrimSpace(h.IssuerID) != "" &&
		strings.TrimSpace(h.PrivateKeyPEM) != ""
}

// AppleIAPConfig App Store 共享密钥（verifyReceipt）。
type AppleIAPConfig struct {
	SharedSecret string `mapstructure:"shared_secret"`
	BundleID     string `mapstructure:"bundle_id"`
}

// Configured 是否具备验票据字段。
func (a AppleIAPConfig) Configured() bool {
	return strings.TrimSpace(a.SharedSecret) != ""
}

// WeChatLoginConfigured 微信 OAuth 换 code 所需字段是否齐全。
func (t ThirdPartyConfig) WeChatLoginConfigured() bool {
	return strings.TrimSpace(t.WeChatAppID) != "" &&
		strings.TrimSpace(t.WeChatAppSecret) != ""
}

// WeChatPayConfigured 微信 App 支付商户字段是否齐全。
func (t ThirdPartyConfig) WeChatPayConfigured() bool {
	return strings.TrimSpace(t.WeChatAppID) != "" &&
		strings.TrimSpace(t.WeChatMchID) != "" &&
		strings.TrimSpace(t.WeChatMchAPIKey) != "" &&
		strings.TrimSpace(t.WeChatMchNotifyURL) != ""
}

// AlipayPayConfigured 支付宝 App 支付签单字段是否齐全。
func (t ThirdPartyConfig) AlipayPayConfigured() bool {
	return strings.TrimSpace(t.AlipayAppID) != "" &&
		strings.TrimSpace(t.AlipayPrivateKey) != ""
}

// HuaweiLoginConfigured 华为账号 OAuth 换 code 所需字段是否齐全。
func (t ThirdPartyConfig) HuaweiLoginConfigured() bool {
	return strings.TrimSpace(t.HuaweiClientID) != "" &&
		strings.TrimSpace(t.HuaweiClientSecret) != ""
}


// JPushConfigured 极光服务端发推送所需字段是否齐全（占位 AppKey 不算）。
func (t ThirdPartyConfig) JPushConfigured() bool {
	key := strings.TrimSpace(t.JPushAppKey)
	secret := strings.TrimSpace(t.JPushMasterSecret)
	if key == "" || secret == "" {
		return false
	}
	return !strings.Contains(strings.ToUpper(key), "PLACEHOLDER")
}
