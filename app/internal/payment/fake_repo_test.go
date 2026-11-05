// fake_repo_test.go 单测用内存仓储：语义对齐 repo_postgres.go，
// 尤其是 WithTx 的回滚（核销 + 发货必须一起成或一起不成）与 SettlePaid 的单向状态机。
package payment

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"model/entity"
)

type fakeRepo struct {
	records map[string]entity.WysPaymentRecord
	diffs   []entity.WysPaymentReconcileDiff
	nextID  int64
	// settleCalls 记录 SettlePaid 被真正改到 paid 的次数（幂等断言用）。
	settleCalls int
	createCalls int
	inTx        int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{records: make(map[string]entity.WysPaymentRecord), nextID: 1}
}

func (f *fakeRepo) seedPending(outTradeNo, channel, refType, refID string, amountFen int64, createdAt time.Time) {
	f.nextID++
	f.records[outTradeNo] = entity.WysPaymentRecord{
		ID:         f.nextID,
		OutTradeNo: outTradeNo,
		UserID:     "user-1",
		Channel:    channel,
		AmountFen:  amountFen,
		Status:     entity.PaymentStatusPending,
		RefType:    refType,
		RefID:      refID,
		NotifyRaw:  "",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
}

func (f *fakeRepo) status(t *testing.T, outTradeNo string) int16 {
	t.Helper()
	rec, ok := f.records[outTradeNo]
	if !ok {
		t.Fatalf("流水 %s 不存在", outTradeNo)
	}
	return rec.Status
}

func (f *fakeRepo) record(outTradeNo string) (entity.WysPaymentRecord, bool) {
	rec, ok := f.records[outTradeNo]
	return rec, ok
}

func (f *fakeRepo) CreatePending(_ context.Context, rec *entity.WysPaymentRecord) error {
	if _, exists := f.records[rec.OutTradeNo]; exists {
		return errors.New("out_trade_no 重复")
	}
	f.createCalls++
	f.nextID++
	rec.ID = f.nextID
	rec.Status = entity.PaymentStatusPending
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	f.records[rec.OutTradeNo] = *rec
	return nil
}

func (f *fakeRepo) GetByOutTradeNo(_ context.Context, outTradeNo string) (*entity.WysPaymentRecord, error) {
	rec, ok := f.records[outTradeNo]
	if !ok {
		return nil, ErrRecordNotFound
	}
	return &rec, nil
}

func (f *fakeRepo) findByRef(refType, refID string, paidOnly bool) (*entity.WysPaymentRecord, error) {
	var matched []entity.WysPaymentRecord
	for _, rec := range f.records {
		if rec.RefType == refType && rec.RefID == refID && (!paidOnly || rec.Status == entity.PaymentStatusPaid) {
			matched = append(matched, rec)
		}
	}
	if len(matched) == 0 {
		return nil, ErrRecordNotFound
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID > matched[j].ID })
	return &matched[0], nil
}

func (f *fakeRepo) GetPaidByRef(_ context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	return f.findByRef(refType, refID, true)
}

func (f *fakeRepo) GetLatestByRef(_ context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	return f.findByRef(refType, refID, false)
}

func (f *fakeRepo) SettlePaid(_ context.Context, outTradeNo, channelTradeNo string, amountFen int64, notifyRaw string, paidAt time.Time) (*entity.WysPaymentRecord, bool, error) {
	rec, ok := f.records[outTradeNo]
	if !ok {
		return nil, false, ErrRecordNotFound
	}
	if rec.Status == entity.PaymentStatusPaid {
		return &rec, false, nil
	}
	if rec.Status == entity.PaymentStatusFailed {
		return nil, false, ErrRecordNotPending
	}
	if rec.AmountFen != amountFen {
		return nil, false, ErrAmountMismatch
	}
	rec.Status = entity.PaymentStatusPaid
	tradeNo := channelTradeNo
	rec.ChannelTradeNo = &tradeNo
	rec.NotifyRaw = notifyRaw
	rec.PaidAt = &paidAt
	f.records[outTradeNo] = rec
	f.settleCalls++
	return &rec, true, nil
}

func (f *fakeRepo) MarkFailed(_ context.Context, outTradeNo, reason string) error {
	rec, ok := f.records[outTradeNo]
	if !ok || rec.Status != entity.PaymentStatusPending {
		return nil
	}
	rec.Status = entity.PaymentStatusFailed
	rec.FailReason = reason
	f.records[outTradeNo] = rec
	return nil
}

func (f *fakeRepo) ListStalePending(_ context.Context, olderThan time.Time, limit int) ([]entity.WysPaymentRecord, error) {
	var rows []entity.WysPaymentRecord
	for _, rec := range f.sortedRecords() {
		if rec.Status == entity.PaymentStatusPending && !rec.CreatedAt.After(olderThan) {
			rows = append(rows, rec)
		}
	}
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeRepo) ListWindow(_ context.Context, channel string, from, to time.Time) ([]entity.WysPaymentRecord, error) {
	var rows []entity.WysPaymentRecord
	for _, rec := range f.sortedRecords() {
		if channel != "" && rec.Channel != channel {
			continue
		}
		if rec.CreatedAt.Before(from) || !rec.CreatedAt.Before(to) {
			continue
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

func (f *fakeRepo) sortedRecords() []entity.WysPaymentRecord {
	rows := make([]entity.WysPaymentRecord, 0, len(f.records))
	for _, rec := range f.records {
		rows = append(rows, rec)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func (f *fakeRepo) CreateDiff(_ context.Context, diff *entity.WysPaymentReconcileDiff) error {
	key := diff.ReconDate + "|" + diff.OutTradeNo + "|" + diff.Kind
	for _, old := range f.diffs {
		if old.ReconDate+"|"+old.OutTradeNo+"|"+old.Kind == key {
			return nil // 对齐唯一索引 + ON CONFLICT DO NOTHING
		}
	}
	f.diffs = append(f.diffs, *diff)
	return nil
}

func (f *fakeRepo) diffCount(outTradeNo string) int {
	n := 0
	for _, d := range f.diffs {
		if d.OutTradeNo == outTradeNo {
			n++
		}
	}
	return n
}

// WithTx 用快照/回滚模拟真库事务：fn 返回 error 时状态整体退回。
func (f *fakeRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	snapshot := make(map[string]entity.WysPaymentRecord, len(f.records))
	for k, v := range f.records {
		snapshot[k] = v
	}
	diffLen := len(f.diffs)
	f.inTx++
	defer func() { f.inTx-- }()
	if err := fn(ctx); err != nil {
		f.records = snapshot
		f.diffs = f.diffs[:diffLen]
		return err
	}
	return nil
}

var _ Repository = (*fakeRepo)(nil)

// fakeSettler 记录业务核销调用；err 非空则模拟发货失败。
type fakeSettler struct {
	calls  []string
	err    error
	onCall func(rec *entity.WysPaymentRecord) error
}

func (s *fakeSettler) SettleChannelPaid(_ context.Context, refType, refID string, rec *entity.WysPaymentRecord) error {
	s.calls = append(s.calls, refType+":"+refID)
	if s.err != nil {
		return s.err
	}
	if s.onCall != nil {
		return s.onCall(rec)
	}
	return nil
}

func wechatXML(vals map[string]string, apiKey string) []byte {
	signed := make(map[string]string, len(vals)+1)
	for k, v := range vals {
		signed[k] = v
	}
	if apiKey != "" {
		signed["sign"] = wechatSign(signed, apiKey)
	}
	// 显式排序，保证报文字节稳定（篡改类用例要按字面量替换节点）。
	keys := make([]string, 0, len(signed))
	for k := range signed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("<xml>")
	for _, k := range keys {
		b.WriteString("<")
		b.WriteString(k)
		b.WriteString("><![CDATA[")
		b.WriteString(signed[k])
		b.WriteString("]]></")
		b.WriteString(k)
		b.WriteString(">")
	}
	b.WriteString("</xml>")
	return []byte(b.String())
}

// mustSettled 断言 NotifyResult 的核销结论。
func mustSettled(t *testing.T, res *NotifyResult, err error, wantNewly bool, wantReason string) {
	t.Helper()
	if err != nil {
		t.Fatalf("期望核销成功，实得 err=%v", err)
	}
	if res == nil {
		t.Fatal("res 为空")
	}
	if res.NewlySettled != wantNewly {
		t.Errorf("NewlySettled=%v，期望 %v", res.NewlySettled, wantNewly)
	}
	if res.Reason != wantReason {
		t.Errorf("Reason=%q，期望 %q", res.Reason, wantReason)
	}
}
