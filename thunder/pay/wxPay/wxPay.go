// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 wxPay
package wxPay

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"errors"

	// 语句或表达式
	"github.com/go-pay/gopay"
	// 语句或表达式
	"github.com/go-pay/gopay/wechat/v3"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 闭合括号（import/var/参数列表等）
)

// 变量声明
var instance *WxPay

// 定义结构体类型 WxPay
type WxPay struct {
	// 语句或表达式
	Client *wechat.ClientV3
	// 语句或表达式
}

// 定义结构体类型 PayBody
type PayBody struct {
	// 语句或表达式
	Description string `json:"description"`
	// 语句或表达式
	OutTradeNo string `json:"out_trade_no"`
	// 语句或表达式
	TimeExpire string `json:"time_expire"`
	// 语句或表达式
	Amount float64 `json:"amount"`
	// 语句或表达式
	OpenId string `json:"openId"`
	// 语句或表达式
	ClientIp string `json:"clientIp"`
	// 语句或表达式
}

// 定义函数 Init（签名开始）
func Init(pay *config.WxPay) {
	// 条件判断 if
	if pay == nil {
		// 返回语句
		return
		// 语句或表达式
	}
	// 条件判断 if
	if pay.MchId == nil {
		// 语句或表达式
		panic("WxPay mchId is nil")
		// 语句或表达式
	}
	// 条件判断 if
	if pay.MchSerialNo == nil {
		// 语句或表达式
		panic("WxPay appId is nil")
		// 语句或表达式
	}
	// 条件判断 if
	if pay.ApiV3Key == nil {
		// 语句或表达式
		panic("WxPay apiV3Key is nil")
		// 语句或表达式
	}
	// 条件判断 if
	if pay.PrivateKey == nil {
		// 语句或表达式
		panic("WxPay privateKey is nil")
		// 语句或表达式
	}
	// 短变量声明并赋值
	client, err := wechat.NewClientV3(*pay.MchId, *pay.MchSerialNo, *pay.ApiV3Key, *pay.PrivateKey)
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 赋值语句
	err = client.AutoVerifySign()
	// 条件判断 if
	if err != nil {
		// 语句或表达式
		panic(err)
		// 语句或表达式
	}
	// 打开Debug开关，输出日志，默认是关闭的
	// 语句或表达式
	client.DebugSwitch = gopay.DebugOn
	//全局实例化 便于使用
	// 赋值语句
	instance = &WxPay{
		// 列表项（字段/参数/元素），行末逗号
		Client: client,
		// 语句或表达式
	}
	// 语句或表达式
}

// 定义函数 Native（签名开始）
func Native(ctx context.Context, body *PayBody) (string, error) {
	// 短变量声明并赋值
	conf := config.GetConfig()
	// 条件判断 if
	if conf.Pay == nil {
		// 语句或表达式
		panic("WxPay config is required")
		// 语句或表达式
	}
	// 短变量声明并赋值
	pay := conf.Pay.WxPay
	// 条件判断 if
	if pay == nil {
		// 返回语句
		return "", errors.New("WxPay config is required")
		// 语句或表达式
	}
	// 短变量声明并赋值
	bm := make(gopay.BodyMap)
	// 语句或表达式
	bm.Set("appid", pay.GetAppId()).
		// 语句或表达式
		Set("mchid", pay.GetMchId()).
		// 语句或表达式
		Set("description", body.Description).
		// 语句或表达式
		Set("out_trade_no", body.OutTradeNo).
		// 语句或表达式
		Set("time_expire", body.TimeExpire).
		// 语句或表达式
		Set("notify_url", pay.GetNotifyUrl()).
		// 语句或表达式
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			// 语句或表达式
			bm.Set("total", body.Amount).
				// 语句或表达式
				Set("currency", "CNY")
			// 语句或表达式
		})
	// 短变量声明并赋值
	rsp, err := instance.Client.V3TransactionNative(ctx, bm)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 返回语句
	return rsp.Response.CodeUrl, nil
	// 语句或表达式
}

// 定义函数 JsApi（签名开始）
func JsApi(ctx context.Context, body *PayBody) (*wechat.JSAPIPayParams, error) {
	// 短变量声明并赋值
	conf := config.GetConfig()
	// 条件判断 if
	if conf.Pay == nil {
		// 语句或表达式
		panic("WxPay config is required")
		// 语句或表达式
	}
	// 短变量声明并赋值
	pay := conf.Pay.WxPay
	// 条件判断 if
	if pay == nil {
		// 返回语句
		return nil, errors.New("WxPay config is required")
		// 语句或表达式
	}

	// 短变量声明并赋值
	bm := make(gopay.BodyMap)
	// 语句或表达式
	bm.Set("appid", pay.GetAppId()).
		// 语句或表达式
		Set("mchid", pay.GetMchId()).
		// 语句或表达式
		Set("description", body.Description).
		// 语句或表达式
		Set("out_trade_no", body.OutTradeNo).
		// 语句或表达式
		Set("time_expire", body.TimeExpire).
		// 语句或表达式
		Set("notify_url", pay.GetNotifyUrl()).
		// 语句或表达式
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			// 语句或表达式
			bm.Set("total", body.Amount).
				// 语句或表达式
				Set("currency", "CNY")
			// 语句或表达式
		}).
		// 语句或表达式
		SetBodyMap("payer", func(bm gopay.BodyMap) {
			// 语句或表达式
			bm.Set("openid", body.OpenId)
			// 语句或表达式
		})
	// 短变量声明并赋值
	rsp, err := instance.Client.V3TransactionJsapi(ctx, bm)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return nil, err
		// 语句或表达式
	}
	// 条件判断 if
	if rsp.Code == 0 {
		// 短变量声明并赋值
		jsapi, err := instance.Client.PaySignOfJSAPI(pay.GetAppId(), rsp.Response.PrepayId)
		// 条件判断 if
		if err != nil {
			// 返回语句
			return nil, err
			// 语句或表达式
		}
		// 返回语句
		return jsapi, nil
		// 语句或表达式
	}
	// 返回语句
	return nil, errors.New("fail")
	// 语句或表达式
}

// 定义函数 H5Pay（签名开始）
func H5Pay(ctx context.Context, body *PayBody) (string, error) {
	// 短变量声明并赋值
	conf := config.GetConfig()
	// 条件判断 if
	if conf.Pay == nil {
		// 语句或表达式
		panic("WxPay config is required")
		// 语句或表达式
	}
	// 短变量声明并赋值
	pay := conf.Pay.WxPay
	// 条件判断 if
	if pay == nil {
		// 返回语句
		return "", errors.New("WxPay config is required")
		// 语句或表达式
	}

	// 短变量声明并赋值
	bm := make(gopay.BodyMap)
	// 语句或表达式
	bm.Set("appid", pay.GetAppId()).
		// 语句或表达式
		Set("mchid", pay.GetMchId()).
		// 语句或表达式
		Set("description", body.Description).
		// 语句或表达式
		Set("out_trade_no", body.OutTradeNo).
		// 语句或表达式
		Set("time_expire", body.TimeExpire).
		// 语句或表达式
		Set("notify_url", pay.GetNotifyUrl()).
		// 语句或表达式
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			// 语句或表达式
			bm.Set("total", body.Amount).
				// 语句或表达式
				Set("currency", "CNY")
			// 语句或表达式
		}).
		// 语句或表达式
		SetBodyMap("scene_info", func(bm gopay.BodyMap) {
			// 语句或表达式
			bm.Set("payer_client_ip", body.ClientIp).
				// 语句或表达式
				SetBodyMap("h5_info", func(bm gopay.BodyMap) {
					// 语句或表达式
					bm.Set("type", "Wap").
						// 语句或表达式
						Set("app_name", "码神之路")
					// 语句或表达式
				})
			// 语句或表达式
		})
	// 短变量声明并赋值
	rsp, err := instance.Client.V3TransactionH5(ctx, bm)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return "", err
		// 语句或表达式
	}
	// 条件判断 if
	if rsp.Code == 0 {
		// 条件判断 if
		if err != nil {
			// 返回语句
			return "", err
			// 语句或表达式
		}
		// 返回语句
		return rsp.Response.H5Url, nil
		// 语句或表达式
	}
	// 返回语句
	return "", errors.New("fail")
	// 语句或表达式
}
