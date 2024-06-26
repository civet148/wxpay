package wxpay

import (
	"context"
	"github.com/civet148/log"
	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"net/http"
)

const PayUnit = 100

type PaymentClient struct {
	cfg     *Config
	client  *core.Client    // 微信支付链接
	handler *notify.Handler // 回调函数的验签+解密
}

type Config struct {
	MchId         string // 商户号
	MchCerSerNum  string // 商户证书序列号
	MchAPIv3Key   string // 商户APIv3秘钥
	MchPrivateKey string // 商户私钥串或PEM私钥文件路径
}

func NewPaymentClient(cfg *Config) *PaymentClient {
	mchPrivateKey := newPrivateKey(cfg.MchPrivateKey)
	m := &PaymentClient{
		cfg:     cfg,
		client:  newPaymentClient(cfg, mchPrivateKey),
		handler: newCallbackHandler(cfg, mchPrivateKey),
	}
	return m
}

// Prepay 	扫码付款
// 返回:     扫码支付链接
func (m *PaymentClient) Prepay(req *PrepayRequest) (strCodeUrl string, err error) {
	svc := native.NativeApiService{Client: m.client}
	resp, result, err := svc.Prepay(context.Background(), native.PrepayRequest{
		Appid:       core.String(req.AppId),
		Mchid:       core.String(m.cfg.MchId),
		Description: core.String(req.Description),
		OutTradeNo:  core.String(req.TradeNo),
		TimeExpire:  core.Time(MinuteAfter(req.ExpireMinutes)),
		NotifyUrl:   core.String(req.NotifyUrl),
		GoodsTag:    core.String(req.GoodsTag),
		LimitPay:    req.LimitPay,
		Amount: &native.Amount{
			Currency: core.String(req.Currency),
			Total:    core.Int64(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()),
		},
		Detail:     req.Detail,
		SettleInfo: req.SettleInfo,
		SceneInfo:  req.SceneInfo,
	})
	if err != nil {
		return "", log.Errorf(err.Error())
	}
	if result.Response.StatusCode != http.StatusOK {
		return "", log.Errorf("response http code is [%v]", result.Response.StatusCode)
	}
	return *resp.CodeUrl, nil
}

func (m *PaymentClient) Refund(req *RefundRequest) (*refunddomestic.Refund, error) {
	svc := refunddomestic.RefundsApiService{Client: m.client}
	resp, result, err := svc.Create(context.Background(), refunddomestic.CreateRequest{
		SubMchid:      core.String(req.SubMchId),
		TransactionId: core.String(req.TransactionId),
		OutTradeNo:    core.String(req.TradeNo),
		OutRefundNo:   core.String(req.RefundNo),
		Reason:        core.String(req.Reason),
		NotifyUrl:     core.String(req.NotifyUrl),
		Amount: &refunddomestic.AmountReq{
			Total:    core.Int64(req.TotalAmount),
			Refund:   core.Int64(req.RefundAmount),
			Currency: core.String(req.Currency),
			From:     nil,
		},
		FundsAccount: nil,
		GoodsDetail:  nil,
	})
	if err != nil {
		return nil, log.Errorf(err.Error())
	}
	if result.Response.StatusCode != http.StatusOK {
		return nil, log.Errorf("response http code is [%v]", result.Response.StatusCode)
	}
	return resp, nil
}

// QueryOrderByTradeNo 通过订单号查询订单状态
func (m *PaymentClient) QueryOrderByTradeNo(strTradeNo string) (tx *payments.Transaction, err error) {
	var result *core.APIResult
	svc := native.NativeApiService{Client: m.client}
	tx, result, err = svc.QueryOrderByOutTradeNo(context.Background(),
		native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(strTradeNo),
			Mchid:      core.String(m.cfg.MchId),
		},
	)
	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}
	if tx == nil || result == nil {
		return nil, log.Errorf("query order response %v or result %v is empty", tx, result)
	}
	if result.Response == nil {
		return nil, log.Errorf("query order result response is empty")
	}
	if result.Response.StatusCode != http.StatusOK {
		err = log.Error("response code is :", result.Response.StatusCode)
		return nil, err
	}
	if tx.TradeState == nil {
		return nil, log.Errorf("query order response trade state is empty")
	}
	return tx, nil
}

// QueryOrderById 通过交易ID查询订单状态
func (m *PaymentClient) QueryOrderById(strTransactionId string) (tx *payments.Transaction, err error) {
	var result *core.APIResult
	svc := native.NativeApiService{Client: m.client}
	tx, result, err = svc.QueryOrderById(context.Background(),
		native.QueryOrderByIdRequest{
			TransactionId: core.String(strTransactionId),
			Mchid:         core.String(m.cfg.MchId),
		},
	)
	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}
	if tx == nil || result == nil {
		return nil, log.Errorf("query order response %v or result %v is empty", tx, result)
	}
	if result.Response == nil {
		return nil, log.Errorf("query order result response is empty")
	}
	if result.Response.StatusCode != http.StatusOK {
		err = log.Error("response code is :", result.Response.StatusCode)
		return nil, err
	}
	if tx.TradeState == nil {
		return nil, log.Errorf("query order response trade state is empty")
	}
	return tx, nil
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

// WxPayNotifyHandler 支付回调处理(如果writer参数为nil则需要手动返回成功消息给微信支付平台)
func (m *PaymentClient) WxPayNotifyHandler(writer http.ResponseWriter, request *http.Request) (tx *payments.Transaction, err error) {
	tx = new(payments.Transaction)
	_, err = m.handler.ParseNotifyRequest(context.Background(), request, tx)
	// 如果验签未通过，或者解密失败
	if err != nil {
		return nil, log.Error("wxpay parse notify request error [%s]", err.Error())
	}
	if writer != nil {
		var resp = NewNotifySuccessResp()
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write(resp.JsonData())
		if err != nil {
			log.Warnf("payment notify ack send failed, error [%s]", err.Error())
			return tx, nil
		}
	}
	return tx, nil
}
