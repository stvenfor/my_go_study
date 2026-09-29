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
	AccessTokenKey = "access_token::"
	// 赋值语句
	RefreshTokenKey = "refresh_token::"
	// 赋值语句
	AccessTokenTTL = 7000
	// 赋值语句
	RefreshTokenTTL = 7000

// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 AccessToken
type AccessToken struct {
	// 语句或表达式
	AccessToken string `json:"access_token"`
	// 语句或表达式
	ExpiresIn int `json:"expires_in"`
	// 语句或表达式
}

// 定义函数 GetAccessToken（签名开始）
func GetAccessToken(appId string, secret string, cache *cache.RedisCache) (string, error) {
	// 短变量声明并赋值
	token, err2 := cache.Get(AccessTokenKey)
	// 条件判断 if
	if err2 == nil && token != "" {
		// 返回语句
		return token, nil
		// 语句或表达式
	}
	// 短变量声明并赋值
	client := httputils.NewHTTPClient(10 * time.Second)
	// 短变量声明并赋值
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appId, secret)
	// 短变量声明并赋值
	headers := map[string]string{}
	// 短变量声明并赋值
	respStr, err := client.GET(url, headers)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("GetAccessToken:", err)
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 变量声明
	var accessToken AccessToken
	// 赋值语句
	err = json.Unmarshal([]byte(respStr), &accessToken)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("json.Unmarshal AccessToken:", err)
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 赋值语句
	err = cache.Set(AccessTokenKey, accessToken.AccessToken, int64(AccessTokenTTL))
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("cache AccessToken:", err)
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 返回语句
	return accessToken.AccessToken, nil
	// 语句或表达式
}

// 定义函数 GetUserInfo（签名开始）
func GetUserInfo(accessToken string, openId string) (*UserInfo, error) {
	// 短变量声明并赋值
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", accessToken, openId)
	// 短变量声明并赋值
	client := httputils.NewHTTPClient(10 * time.Second)
	// 短变量声明并赋值
	headers := map[string]string{}
	// 短变量声明并赋值
	respStr, err := client.GET(url, headers)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		log.Println("GetUserInfo:", err)
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
