package wxpay

import (
	"encoding/json"
	"github.com/shopspring/decimal"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

const (
	privateKeyBegin = "-----BEGIN PRIVATE KEY-----"
	privateKeyEnd   = "-----END PRIVATE KEY-----"
)

type TradeState string

const (
	TradeState_Success  TradeState = "SUCCESS"  //成功
	TradeState_Accepted TradeState = "ACCEPTED" //处理中
	TradeState_PayFail  TradeState = "PAY_FAIL" //支付失败
	TradeState_Refund   TradeState = "REFUND"   //已退款
)

type NotifyResp struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewNotifySuccessResp() *NotifyResp {
	return &NotifyResp{
		Code:    "0",
		Message: "SUCCESS",
	}
}

func (r NotifyResp) JsonData() []byte {
	data, _ := json.Marshal(&r)
	return data
}

// 扫码支付请求
type PrepayRequest struct {
	//[*] 应用ID
	AppId string
	//[*] 支付金额(单位: 元)
	Amount decimal.Decimal
	//[*] 符合ISO 4217标准的三位字母代码，目前只支持人民币：CNY。
	Currency string
	//[*] 商户订单号
	TradeNo string
	//[*] 订单失效时间(分钟)
	ExpireMinutes int
	// 回调通知URL，必须是HTTPS且不允许携带查询串
	NotifyUrl string
	// 商品描述
	Description string
	// 附加数据
	Attach string
	// 商品标记，代金券或立减优惠功能的参数。
	GoodsTag string
	// 指定支付方式
	LimitPay []string
	// 传入true时，支付成功消息和支付详情页将出现开票入口。需要在微信支付商户平台或微信公众平台开通电子发票功能，传此字段才可生效。
	SupportFapiao bool

	Detail     *native.Detail
	SettleInfo *native.SettleInfo
	SceneInfo  *native.SceneInfo
}

// 退款请求
type RefundRequest struct {
	//[*] 商户订单号(跟TransactionId二选一)
	TradeNo string

	//[*] 原支付交易对应的微信订单号(跟TradeNo二选一)
	TransactionId string

	//[*] 商户系统内部的退款单号，商户系统内部唯一，只能是数字、大小写字母_-|*@ ，同一退款单号多次请求只退一笔。
	RefundNo string

	//[*] 退款金额(单位: 分)，不能超过原订单支付金额。
	RefundAmount int64

	//[*] 原支付交易的订单总金额(单位: 分)
	TotalAmount int64

	//[*] 符合ISO 4217标准的三位字母代码，目前只支持人民币：CNY。
	Currency string

	// 子商户的商户号，由微信支付生成并下发。服务商模式下必须传递此参数
	SubMchId string

	// 异步接收微信支付退款结果通知的回调地址，通知url必须为外网可访问的url，不能携带参数。 如果参数中传了notify_url，则商户平台上配置的回调地址将不会生效，优先回调当前传的这个地址。
	NotifyUrl string

	// 若商户传入，会在下发给用户的退款消息中体现退款原因
	Reason string
}
