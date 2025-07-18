package wxpay

import (
	"context"
	"crypto/rsa"
	"fmt"
	"github.com/civet148/log"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"strings"
)

const (
	PemSuffix = ".pem"
)

// newPrivateKey load from pem file (suffix .pem) or private key string
func newPrivateKey(strPrivateKey string) (mchPrivateKey *rsa.PrivateKey) {
	var err error
	if strPrivateKey == "" {
		panic("pem file path or private key string is empty")
	}
	if strings.HasSuffix(strPrivateKey, PemSuffix) {
		mchPrivateKey, err = utils.LoadPrivateKeyWithPath(strPrivateKey)
	} else {
		strPrivateKey = completePrivateKey(strPrivateKey)
		mchPrivateKey, err = utils.LoadPrivateKey(strPrivateKey)
	}
	if err != nil {
		log.Panic("load merchant private key error")
	}
	return mchPrivateKey
}

func newPaymentClient(cfg *Config, mchPrivateKey *rsa.PrivateKey) (client *core.Client) {
	var err error
	ctx := context.Background()
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(cfg.MchId, cfg.MchCerSerNum, mchPrivateKey, cfg.MchAPIv3Key),
	}
	client, err = core.NewClient(ctx, opts...)
	if err != nil {
		log.Panic("new wechat pay client err:", err)
	}
	return client
}

func newCallbackHandler(cfg *Config, mchPrivateKey *rsa.PrivateKey) (handler *notify.Handler) {
	ctx := context.Background()
	// 1. 使用 `RegisterDownloaderWithPrivateKey` 注册下载器
	err := downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, mchPrivateKey, cfg.MchCerSerNum, cfg.MchId, cfg.MchAPIv3Key)
	if err != nil {
		log.Panic("new downer handler err:", err)
	}
	// 2. 获取商户号对应的微信支付平台证书访问器
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(cfg.MchId)
	// 3. 使用证书访问器初始化 `notify.Handler`
	handler = notify.NewNotifyHandler(cfg.MchAPIv3Key, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	return handler
}

func completePrivateKey(strPrivateKey string) string {
	if !strings.Contains(strPrivateKey, privateKeyBegin) {
		strPrivateKey = fmt.Sprintf("%s\n%s", privateKeyBegin, strPrivateKey)
	}
	if !strings.Contains(strPrivateKey, privateKeyEnd) {
		strPrivateKey = fmt.Sprintf("%s\n%s", strPrivateKey, privateKeyEnd)
	}
	return strPrivateKey
}
