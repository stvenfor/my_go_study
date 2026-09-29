// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 wx
package wx

// import 多行导入块开始
import (
	// 语句或表达式
	"encoding/json"
	// 语句或表达式
	"errors"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"github.com/mszlu521/thunder/cache"
	// 语句或表达式
	"github.com/mszlu521/thunder/tools/httputils"
	// 语句或表达式
	"log"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// var 变量块开始
var (
	// 赋值语句
	WebAccessTokenKey = "web_access_token::"
	// 赋值语句
	WebAccessTokenTTL = 7000
	// 赋值语句
	WebRefreshTokenKey = "web_refresh_token::"
	// 赋值语句
	WebRefreshTokenTTL = 3600 * 24 * 29

// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 WebAccessToken
type WebAccessToken struct {
	// 语句或表达式
	AccessToken string `json:"access_token"`
	// 语句或表达式
	ExpiresIn int `json:"expires_in"`
	// 语句或表达式
	RefreshToken string `json:"refresh_token"`
	// 语句或表达式
	Openid string `json:"openid"`
	// 语句或表达式
	Scope string `json:"scope"`
	// 语句或表达式
	Unionid string `json:"unionid"`
	// 语句或表达式
	Errcode int `json:"errcode"`
	// 语句或表达式
	Errmsg string `json:"errmsg"`
	// 语句或表达式
}

// 定义函数 GetAccessTokenByCode（签名开始）
func GetAccessTokenByCode(appId string, secret string, code string, cache *cache.RedisCache) (string, string, error) {
	// 短变量声明并赋值
	client := httputils.NewHTTPClient(10 * time.Second)
	// 短变量声明并赋值
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", appId, secret, code)
	// 短变量声明并赋值
	headers := map[string]string{}
	// 短变量声明并赋值
	respStr, err := client.GET(url, headers)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("GetAccessToken:", err)
		// 返回语句
		return "", "", err
		// 语句或表达式
	}
	// 返回语句
	return dealAccessTokenResponse(respStr, cache)
	// 语句或表达式
}

// 定义函数 GetAccessTokenByCache（签名开始）
func GetAccessTokenByCache(appId string, cache *cache.RedisCache) (string, string, error) {
	// 短变量声明并赋值
	client := httputils.NewHTTPClient(10 * time.Second)
	// 短变量声明并赋值
	token, err2 := cache.Get(AccessTokenKey)
	// 条件判断 if
	if err2 == nil && token != "" {
		// 返回语句
		return token, "", nil
		// 语句或表达式
	}
	// 短变量声明并赋值
	refreshToken, err2 := cache.Get(WebRefreshTokenKey)
	// 条件判断 if
	if err2 == nil && refreshToken != "" {
		// 短变量声明并赋值
		url := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/refresh_token?appid=%s&grant_type=refresh_token&refresh_token=%s", appId, refreshToken)
		// 短变量声明并赋值
		headers := map[string]string{}
		// 短变量声明并赋值
		respStr, err := client.GET(url, headers)
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			log.Println("Get Refresh AccessToken:", err)
			// 返回语句
			return "", "", err
			// 语句或表达式
		}
		// 返回语句
		return dealAccessTokenResponse(respStr, cache)
		// 语句或表达式
	}
	// 返回语句
	return "", "", errors.New("no accessToken")
	// 语句或表达式
}

// 定义函数 dealAccessTokenResponse（签名开始）
func dealAccessTokenResponse(respStr string, cache *cache.RedisCache) (string, string, error) {
	// 变量声明
	var accessToken WebAccessToken
	// 短变量声明并赋值
	err := json.Unmarshal([]byte(respStr), &accessToken)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("json.Unmarshal AccessToken:", err)
		// 返回语句
		return "", "", err
		// 语句或表达式
	}
	// 条件判断 if
	if accessToken.Errcode != 0 {
		//有错误
		// 返回语句
		return "", "", errors.New(accessToken.Errmsg)
		// 语句或表达式
	}
	//在调用其他接口的时候 就不需要重新授权了，直接从缓存获取accessToken
	// 赋值语句
	err = cache.Set(WebAccessTokenKey, accessToken.AccessToken, int64(AccessTokenTTL))
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("cache AccessToken:", err)
		// 返回语句
		return "", "", err
		// 语句或表达式
	}
	// 赋值语句
	err = cache.Set(WebRefreshTokenKey, accessToken.RefreshToken, int64(RefreshTokenTTL))
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("cache refresh AccessToken:", err)
		// 返回语句
		return "", "", err
		// 语句或表达式
	}
	// 返回语句
	return accessToken.AccessToken, accessToken.Openid, nil
	// 语句或表达式
}

// 定义结构体类型 UserInfo
type UserInfo struct {
	// 语句或表达式
	Openid string `json:"openid"`
	// 语句或表达式
	Nickname string `json:"nickname"`
	// 语句或表达式
	Sex int `json:"sex"`
	// 语句或表达式
	Province string `json:"province"`
	// 语句或表达式
	City string `json:"city"`
	// 语句或表达式
	Country string `json:"country"`
	// 语句或表达式
	Headimgurl string `json:"headimgurl"`
	// 语句或表达式
	Privilege []string `json:"privilege"`
	// 语句或表达式
	Unionid string `json:"unionid"`
	// 语句或表达式
	Errcode int `json:"errcode"`
	// 语句或表达式
	Errmsg string `json:"errmsg"`
	// 语句或表达式
}

// 定义函数 GetWebUserInfo（签名开始）
func GetWebUserInfo(token string, openid string) (*UserInfo, error) {
	// 短变量声明并赋值
	client := httputils.NewHTTPClient(10 * time.Second)
	// 短变量声明并赋值
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s", token, openid)
	// 短变量声明并赋值
	headers := map[string]string{}
	// 短变量声明并赋值
	respStr, err := client.GET(url, headers)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("Get Refresh AccessToken:", err)
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 变量声明
	var userInfo UserInfo
	// 赋值语句
	err = json.Unmarshal([]byte(respStr), &userInfo)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("json.Unmarshal UserInfo:", err)
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 条件判断 if
	if userInfo.Errcode != 0 {
		// 语句或表达式
		log.Println("GetUserInfo:", userInfo.Errmsg)
		// 返回语句
		return nil, errors.New(userInfo.Errmsg)
		// 语句或表达式
	}
	// 返回语句
	return &userInfo, nil
	// 语句或表达式
}
