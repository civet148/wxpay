package wxpay

import (
	"context"
	"github.com/civet148/log"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"net/http"
	"strings"
)

const PayUnit = 100

type PaymentClient struct {
	cfg     *Config
	client  *core.Client    // 微信支付链接
	handler *notify.Handler // 回调函数的验签+解密
}

type Config struct {
	MchId        string `json:"mch_id"`          // 商户号
	MchCerSerNum string `json:"mch_cer_ser_num"` // 商户证书序列号
	MchAPIv3Key  string `json:"mch_api_key"`     // 商户APIv3秘钥
	PemPath      string `json:"pem_path"`        // 商户PEM密钥文件路径
}

func NewWechatClient(cfg *Config) *PaymentClient {
	mchPrivateKey := newPrivateKey(cfg.PemPath)
	m := &PaymentClient{
		cfg:     cfg,
		client:  newPaymentClient(cfg, mchPrivateKey),
		handler: newCallbackHandler(cfg, mchPrivateKey),
	}
	return m
}

// Prepay 订单预付款
// wechatAppId    微信应用ID (登录微信开发平台创建并获取应用ID https://mp.weixin.qq.com)
// strDescription 订单描述
// strTradeNo     订单号
// strNotifyUrl   支付回调通知URL(例如：POST https://www.your-company.com/notify/wxpay)
// expireMinutes  订单支付有效时间(分钟)
// payAmount      支付金额(单位: 元)
func (m *PaymentClient) Prepay(wechatAppId, strDescription, strTradeNo, strNotifyUrl string, expireMinutes int, payAmount float64) (strCodeUrl string, err error) {
	svc := native.NativeApiService{Client: m.client}
	resp, result, err := svc.Prepay(context.Background(), native.PrepayRequest{
		Appid:       core.String(wechatAppId),
		Mchid:       core.String(m.cfg.MchId),
		Description: core.String(strDescription),
		OutTradeNo:  core.String(strTradeNo),
		TimeExpire:  core.Time(MinuteAfter(expireMinutes)),
		NotifyUrl:   core.String(strNotifyUrl),
		Amount: &native.Amount{
			Currency: core.String("CNY"),
			Total:    core.Int64(int64(payAmount * PayUnit)), // 转换为分
		},
	})
	if err != nil {
		log.Errorf(err.Error())
		return "", err
	}
	if result.Response.StatusCode != http.StatusOK {
		err = log.Errorf("response http code is [%v]", result.Response.StatusCode)
		return "", err
	}
	return *resp.CodeUrl, nil
}

// QueryOrderByTradeNo 通过订单号查询订单状态
func (m *PaymentClient) QueryOrderByTradeNo(strTradeNo string) (state, message string, err error) {
	svc := native.NativeApiService{Client: m.client}
	resp, result, err := svc.QueryOrderByOutTradeNo(context.Background(),
		native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(strTradeNo),
			Mchid:      core.String(m.cfg.MchId),
		},
	)
	if err != nil {
		log.Errorf(err.Error())
		return "", *resp.TradeStateDesc, err
	}
	if resp == nil || result == nil {
		return "", "", log.Errorf("query order response %v or result %v is empty", resp, result)
	}
	if result.Response == nil {
		return "", "", log.Errorf("query order result response is empty")
	}
	if result.Response.StatusCode != http.StatusOK {
		err = log.Error("response code is :", result.Response.StatusCode)
		return "", *resp.TradeStateDesc, err
	}
	if resp.TradeState == nil {
		return "", "", log.Errorf("query order response trade state is empty")
	}
	if strings.EqualFold(*resp.TradeState, "success") {
		return *resp.TradeState, *resp.TradeStateDesc, nil
	}
	return *resp.TradeState, *resp.TradeStateDesc, nil
}

// QueryOrderById 通过交易ID查询订单状态
func (m *PaymentClient) QueryOrderById(strTransactionId string) (state, message string, err error) {
	svc := native.NativeApiService{Client: m.client}
	resp, result, err := svc.QueryOrderById(context.Background(),
		native.QueryOrderByIdRequest{
			TransactionId: core.String(strTransactionId),
			Mchid:         core.String(m.cfg.MchId),
		},
	)
	if err != nil {
		log.Errorf(err.Error())
		return "", *resp.TradeStateDesc, err
	}
	if resp == nil || result == nil {
		return "", "", log.Errorf("query order response %v or result %v is empty", resp, result)
	}
	if result.Response == nil {
		return "", "", log.Errorf("query order result response is empty")
	}
	if result.Response.StatusCode != http.StatusOK {
		err = log.Error("response code is :", result.Response.StatusCode)
		return "", *resp.TradeStateDesc, err
	}
	if resp.TradeState == nil {
		return "", "", log.Errorf("query order response trade state is empty")
	}
	if strings.EqualFold(*resp.TradeState, "success") {
		return *resp.TradeState, *resp.TradeStateDesc, nil
	}
	return *resp.TradeState, *resp.TradeStateDesc, nil
}

// CloseOrder 关闭支付订单
func (m *PaymentClient) CloseOrder(strTradeNo string) (ok bool, err error) {
	svc := native.NativeApiService{Client: m.client}
	result, err := svc.CloseOrder(context.Background(), native.CloseOrderRequest{
		OutTradeNo: core.String(strTradeNo),
		Mchid:      core.String(m.cfg.MchId),
	})
	if err != nil {
		return false, log.Errorf("request close order error [%s]", err)
	}
	if result.Response.StatusCode != http.StatusNoContent {
		return false, log.Errorf("response code is %v", result.Response.StatusCode)
	}
	return true, nil
}

// WxPayNotifyHandler 支付回调处理
func (m *PaymentClient) WxPayNotifyHandler(request *http.Request) (tx *payments.Transaction, err error) {
	tx = new(payments.Transaction)
	_, err = m.handler.ParseNotifyRequest(context.Background(), request, tx)
	// 如果验签未通过，或者解密失败
	if err != nil {
		return nil, log.Error("wxpay parse notify request error [%s]", err.Error())
	}
	return tx, nil
}
