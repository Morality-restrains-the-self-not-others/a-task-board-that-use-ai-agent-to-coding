package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"confload"
)

const wechatPointsSourceType = "user_recharge_wechat"

type wechatFapiaoConfig struct {
	Enabled       bool   `yaml:"enabled"`
	TaxCode       string `yaml:"tax_code"`
	TaxRate       int    `yaml:"tax_rate"`
	GoodsName     string `yaml:"goods_name"`
	GoodsCategory string `yaml:"goods_category"`
	NotifyURL     string `yaml:"notify_url"`
}

type wechatPayConfig struct {
	Mode                   string             `yaml:"mode"`
	Mchid                  string             `yaml:"mchid"`
	Appid                  string             `yaml:"appid"`
	MerchantSerialNo       string             `yaml:"merchant_serial_no"`
	APIv3Key               string             `yaml:"api_v3_key"`
	MerchantPrivateKeyPath string             `yaml:"merchant_private_key_path"`
	PubKeyID               string             `yaml:"pub_key_id"`
	PubKeyPath             string             `yaml:"pub_key_path"`
	NotifyURL              string             `yaml:"notify_url"`
	DescriptionPrefix      string             `yaml:"description_prefix"`
	Fapiao                 wechatFapiaoConfig `yaml:"fapiao"`
}

type wechatPendingOrder struct {
	OutTradeNo  string
	TenantID    int64
	UserID      string
	AmountYuan  int64
	AmountFen   int64
	Description string
	OrderID     int64 // 资源订单 ID，非零时支付成功后发放资源而非充值余额
	CreatedAt   time.Time
}

var (
	wechatCfg     wechatPayConfig
	wechatDir     string
	wechatClient  *core.Client
	wechatNotifyH *notify.Handler
	wechatMu      sync.Mutex
	wechatPending = map[string]wechatPendingOrder{}
	wechatLiveOK  bool
)

func wechatConfigured() bool {
	if wechatIsMock() {
		return true
	}
	return wechatLiveOK
}

func wechatIsMock() bool {
	return strings.EqualFold(strings.TrimSpace(wechatCfg.Mode), "mock")
}

func loadWechatPayConfig(repoRoot string) {
	wechatDir = filepath.Join(repoRoot, "conf", "billing", "wechatPay")
	// OPT-20260901-012: 经 confload 深合并 conf-local/<同相对路径>，取代对 struct
	// 二次 Unmarshal 的自定义 overlay（零值/浅合并会盖掉 nested 键）。模板在
	// 合并后统一解析，conf-local 中的 ${...} 也能展开。
	if err := confload.UnmarshalYAMLMerged(repoRoot, "billing/wechatPay/conf.yaml", &wechatCfg); err != nil {
		log.Printf("[taskBill] wechatPay conf.yaml missing: %v — WeChat pay disabled", err)
		wechatCfg.Mode = "disabled"
		wechatLiveOK = false
		return
	}
	applyWechatEnvOverrides()
	if wechatCfg.PubKeyPath == "" {
		wechatCfg.PubKeyPath = "pub_key.pem"
	}
	if wechatCfg.DescriptionPrefix == "" {
		wechatCfg.DescriptionPrefix = "平台支付"
	}
	mode := strings.ToLower(strings.TrimSpace(wechatCfg.Mode))
	if mode == "" {
		mode = "live"
	}
	wechatCfg.Mode = mode

	if mode == "live" {
		if err := initWechatLiveClient(); err != nil {
			log.Printf("[taskBill] error: WeChat live init failed (%v) — pay disabled (no mock fallback)", err)
			wechatLiveOK = false
		} else {
			wechatLiveOK = true
			log.Printf("[taskBill] WeChat Pay live client ready (pubkey_id=%s)", maskMid(wechatCfg.PubKeyID))
		}
		return
	}
	if mode == "mock" {
		wechatLiveOK = false
		log.Printf("[taskBill] WeChat Pay mode=mock (pubkey_id=%s)", maskMid(wechatCfg.PubKeyID))
		return
	}
	wechatLiveOK = false
	log.Printf("[taskBill] WeChat Pay mode=%s — disabled", mode)
}

func wechatResolveDataFile(name string) string {
	inConf := filepath.Join(wechatDir, name)
	if _, err := os.Stat(inConf); err == nil {
		return inConf
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(wechatDir)))
	inLocal := filepath.Join(root, "conf-local", "billing", "wechatPay", name)
	if _, err := os.Stat(inLocal); err == nil {
		return inLocal
	}
	return inConf
}

func applyWechatEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_MODE")); v != "" {
		wechatCfg.Mode = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_APPID")); v != "" {
		wechatCfg.Appid = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_MCHID")); v != "" {
		wechatCfg.Mchid = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_MERCHANT_SERIAL_NO")); v != "" {
		wechatCfg.MerchantSerialNo = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_API_V3_KEY")); v != "" {
		wechatCfg.APIv3Key = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_NOTIFY_URL")); v != "" {
		wechatCfg.NotifyURL = v
	}
	if v := strings.TrimSpace(os.Getenv("WECHAT_PAY_PRIVATE_KEY_PATH")); v != "" {
		wechatCfg.MerchantPrivateKeyPath = v
	}
}

func maskMid(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}

func initWechatLiveClient() error {
	if strings.TrimSpace(wechatCfg.Mchid) == "" ||
		strings.TrimSpace(wechatCfg.Appid) == "" ||
		strings.TrimSpace(wechatCfg.MerchantSerialNo) == "" ||
		strings.TrimSpace(wechatCfg.APIv3Key) == "" {
		return fmt.Errorf("missing mchid/appid/merchant_serial_no/api_v3_key")
	}
	keyPath := wechatCfg.MerchantPrivateKeyPath
	if !filepath.IsAbs(keyPath) {
		keyPath = wechatResolveDataFile(keyPath)
	}
	priv, err := utils.LoadPrivateKeyWithPath(keyPath)
	if err != nil {
		return fmt.Errorf("load merchant private key: %w", err)
	}
	pubPath := wechatCfg.PubKeyPath
	if !filepath.IsAbs(pubPath) {
		pubPath = wechatResolveDataFile(pubPath)
	}
	pub, err := utils.LoadPublicKeyWithPath(pubPath)
	if err != nil {
		return fmt.Errorf("load wechat public key: %w", err)
	}
	if strings.TrimSpace(wechatCfg.PubKeyID) == "" {
		return fmt.Errorf("missing pub_key_id")
	}
	ctx := context.Background()
	opts := []core.ClientOption{
		option.WithWechatPayPublicKeyAuthCipher(
			wechatCfg.Mchid,
			wechatCfg.MerchantSerialNo,
			priv,
			wechatCfg.PubKeyID,
			pub,
		),
	}
	client, err := core.NewClient(ctx, opts...)
	if err != nil {
		return err
	}
	wechatClient = client

	handler, err := newWechatNotifyHandler(pub)
	if err != nil {
		return err
	}
	wechatNotifyH = handler
	return nil
}

func newWechatNotifyHandler(pub *rsa.PublicKey) (*notify.Handler, error) {
	key := []byte(wechatCfg.APIv3Key)
	if len(key) != 32 {
		return nil, fmt.Errorf("api_v3_key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	verifier := verifiers.NewSHA256WithRSAPubkeyVerifier(wechatCfg.PubKeyID, *pub)
	h := notify.NewEmptyHandler()
	h.AddRSAWithAESGCM(verifier, aesgcm)
	return h, nil
}

func storeWechatPending(o wechatPendingOrder) {
	wechatMu.Lock()
	wechatPending[o.OutTradeNo] = o
	wechatMu.Unlock()
	// 持久化（v64: 替代内存 map — 进程重启后回调仍可处理）
	_, err := db.Exec(`
		INSERT INTO billing_payment_pending (out_trade_no, user_id, tenant_id, order_id, amount_fen, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?)
		ON DUPLICATE KEY UPDATE
			user_id = VALUES(user_id), tenant_id = VALUES(tenant_id),
			order_id = VALUES(order_id), amount_fen = VALUES(amount_fen)`,
		o.OutTradeNo, o.UserID, o.TenantID, o.OrderID, o.AmountFen,
		o.CreatedAt.UTC().Format("2006-01-02 15:04:05.000000"))
	if err != nil {
		log.Printf("[taskBill] persist wechat pending %s failed: %v", o.OutTradeNo, err)
	}
}

func getWechatPending(outTradeNo string) (wechatPendingOrder, bool) {
	wechatMu.Lock()
	o, ok := wechatPending[outTradeNo]
	wechatMu.Unlock()
	if ok {
		return o, true
	}
	// DB 兜底（进程重启后内存 map 已清空）
	var o2 wechatPendingOrder
	var createdAt string
	err := db.QueryRow(`
		SELECT out_trade_no, user_id, tenant_id, order_id, amount_fen, created_at
		FROM billing_payment_pending WHERE out_trade_no = ?`, outTradeNo,
	).Scan(&o2.OutTradeNo, &o2.UserID, &o2.TenantID, &o2.OrderID, &o2.AmountFen, &createdAt)
	if err != nil {
		return o2, false
	}
	if t, err := time.Parse("2006-01-02 15:04:05.000000", createdAt); err == nil {
		o2.CreatedAt = t
	}
	return o2, true
}

// markWechatPendingPaid 标记持久化 pending 为已支付（幂等）。
func markWechatPendingPaid(outTradeNo string, now time.Time) {
	_, err := db.Exec(`
		UPDATE billing_payment_pending SET status = 'paid', paid_at = ?
		WHERE out_trade_no = ? AND status != 'paid'`,
		now.UTC().Format("2006-01-02 15:04:05.000000"), outTradeNo)
	if err != nil {
		log.Printf("[taskBill] mark wechat pending paid %s failed: %v", outTradeNo, err)
	}
}

func wechatTxnID(outTradeNo string) string {
	id := "wechat:" + outTradeNo
	if len(id) > 100 {
		return id[:100]
	}
	return id
}

// outTradeNoFingerprint 对外单号做短指纹，日志中避免回显完整商户单号（元规则 24）。
func outTradeNoFingerprint(outTradeNo string) string {
	sum := sha256.Sum256([]byte(outTradeNo))
	return fmt.Sprintf("%x", sum[:4])
}

// Native 预下单与金额换算见 wechat_pay_prepay.go / wechat_pay_amount.go。

func wechatCreditFromPending(ctx context.Context, outTradeNo string, totalFen int64) (map[string]interface{}, error) {
	pending, ok := getWechatPending(outTradeNo)
	if !ok {
		return nil, fmt.Errorf("pending order not found")
	}
	if totalFen > 0 && totalFen != pending.AmountFen {
		return nil, fmt.Errorf("amount mismatch: expected %d fen got %d", pending.AmountFen, totalFen)
	}
	// 资源订单支付：发放资源配额而非充值余额
	if pending.OrderID != 0 {
		// OPT-20260819-004: Native code_url 最长约 2 小时仍可支付；仅改预下单挡不住
		// 修复前二维码。回调入账前核对 pending.amount_fen 与订单 total_yuan_cents，
		// 不一致拒绝入账（warn 含 out_trade_no 指纹与两金额），避免多收/错收。
		order, _, err := loadOrder(pending.TenantID, pending.OrderID)
		if err != nil {
			log.Printf("[taskBill] wechat credit load order failed out_trade_no_fp=%s order=%d tenant=%d err=%v",
				outTradeNoFingerprint(outTradeNo), pending.OrderID, pending.TenantID, err)
			return nil, fmt.Errorf("load resource order %d: %w", pending.OrderID, err)
		}
		if order.TotalYuanCents != pending.AmountFen {
			log.Printf("[taskBill] wechat pending amount mismatch out_trade_no_fp=%s order=%d pending_fen=%d order_fen=%d — 拒绝入账",
				outTradeNoFingerprint(outTradeNo), pending.OrderID, pending.AmountFen, order.TotalYuanCents)
			return nil, fmt.Errorf("pending amount %d fen != order total %d fen (order %d)",
				pending.AmountFen, order.TotalYuanCents, pending.OrderID)
		}
		if err := markOrderPaid(ctx, pending.OrderID, "wechat", wechatTxnID(outTradeNo), pending.TenantID); err != nil {
			return nil, fmt.Errorf("fulfill resource order: %w", err)
		}
		persistWechatPayVouchers(pending.OrderID, outTradeNo, "")
		return map[string]interface{}{
			"status":    "success",
			"order_id":  formatID(pending.OrderID),
			"fulfilled": true,
		}, nil
	}
	// 余额充值路径已移除（2026-07-26）；所有支付须通过资源订单路径
	return nil, fmt.Errorf("balance recharge removed — use resource orders")
}

func handleWechatNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"code": "FAIL", "message": "method"})
		return
	}
	if wechatIsMock() {
		body, _ := readJSONBody(r)
		outTradeNo := stringField(body, "out_trade_no")
		if outTradeNo == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "out_trade_no"})
			return
		}
		result, err := wechatCreditFromPending(r.Context(), outTradeNo, 0)
		if err != nil {
			log.Printf("[taskBill] wechat mock notify credit failed: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "credit"})
			return
		}
		orderIDStr, _ := result["order_id"].(string)
		if orderIDStr != "" {
			recordOrderPayCredentials(orderIDStr, "wechat", "web", "", stringField(body, "openid"), stringField(body, "unionid"))
			persistWechatPayVouchersFromNotify(orderIDStr, outTradeNo, stringField(body, "transaction_id"))
		}
		markWechatPendingPaid(outTradeNo, time.Now())
		publishPaymentSucceeded(r.Context(), outTradeNo, orderIDStr, result, stringField(body, "openid"))
		log.Printf("[taskBill] wechat mock notify credited out_trade_no=%s", outTradeNo)
		writeJSON(w, http.StatusOK, map[string]interface{}{"code": "SUCCESS", "message": "成功", "result": result})
		return
	}
	if wechatNotifyH == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"code": "FAIL", "message": "not ready"})
		return
	}
	var txn payments.Transaction
	_, err := wechatNotifyH.ParseNotifyRequest(r.Context(), r, &txn)
	if err != nil {
		log.Printf("[taskBill] wechat notify verify/decrypt failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "verify"})
		return
	}
	if txn.OutTradeNo == nil || *txn.OutTradeNo == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "no out_trade_no"})
		return
	}
	if txn.TradeState != nil && *txn.TradeState != "SUCCESS" {
		writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS", "message": "忽略非成功状态"})
		return
	}
	var totalFen int64
	if txn.Amount != nil && txn.Amount.Total != nil {
		totalFen = *txn.Amount.Total
	}
	result, err := wechatCreditFromPending(r.Context(), *txn.OutTradeNo, totalFen)
	if err != nil {
		log.Printf("[taskBill] wechat notify credit failed out_trade_no=%s err=%v", *txn.OutTradeNo, err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "credit"})
		return
	}
	// 支付凭据记录（审计/对账；登录/支付解耦 — 不判定不拒付）
	orderIDStr, _ := result["order_id"].(string)
	openID := ""
	if txn.Payer != nil && txn.Payer.Openid != nil {
		openID = *txn.Payer.Openid
	}
	appID := ""
	if txn.Appid != nil {
		appID = *txn.Appid
	}
	if orderIDStr != "" {
		recordOrderPayCredentials(orderIDStr, "wechat", "web", appID, openID, "")
		txnID := ""
		if txn.TransactionId != nil {
			txnID = *txn.TransactionId
		}
		persistWechatPayVouchersFromNotify(orderIDStr, *txn.OutTradeNo, txnID)
	}
	markWechatPendingPaid(*txn.OutTradeNo, time.Now())
	publishPaymentSucceeded(r.Context(), *txn.OutTradeNo, orderIDStr, result, openID)
	log.Printf("[taskBill] wechat notify credited out_trade_no=%s", *txn.OutTradeNo)
	writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS", "message": "成功"})
}

// handleRechargeWechatCreatePublic 处理微信支付创建（余额充值路径）。
// 前端已无独立充值页面，但管理员授权、推荐佣金、PayPal 回调等仍通过本路径为账户充值余额。
// 用户下单购买资源走 handlers_orders.go handlePayOrder 路径。

// recordOrderPayCredentials 记录支付凭据到订单（审计/对账用；登录/支付解耦 — 不判定）。
func recordOrderPayCredentials(orderID, payMethod, appKey, appID, openID, unionID string) {
	if orderID == "" {
		return
	}
	if _, err := db.Exec(`
		UPDATE billing_resource_order
		SET pay_method = ?, pay_app_key = ?, pay_openid = ?, pay_unionid = ?
		WHERE id = ?`, payMethod, appKey, openID, unionID, orderID); err != nil {
		log.Printf("[taskBill] record order pay credentials order=%s failed: %v", orderID, err)
	}
}

// publishPaymentSucceeded 发布 PAYMENT_SUCCEEDED 事件（回调入账成功后，异步）。
func publishPaymentSucceeded(ctx context.Context, outTradeNo, orderIDStr string, result map[string]interface{}, openID string) {
	userID, _ := result["user_id"].(string)
	tenantID, _ := result["tenant_id"].(string)
	payload := map[string]interface{}{
		"out_trade_no": outTradeNo,
		"order_id":     orderIDStr,
		"user_id":      userID,
		"tenant_id":    tenantID,
		"pay_method":   "wechat",
		"pay_openid":   openID,
	}
	go func() {
		if err := publishEvent(context.Background(), "PAYMENT_SUCCEEDED", payload, outTradeNo); err != nil {
			log.Printf("[taskBill] publish PAYMENT_SUCCEEDED %s failed: %v", outTradeNo, err)
		}
	}()
}
