package service

import (
	"time"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
}

type AdminOverview struct {
	TradingSymbols      int64           `json:"trading_symbols"`
	OpenPositions       int64           `json:"open_positions"`
	AccountsAtRisk      int64           `json:"accounts_at_risk"`
	AccountsReduceOnly  int64           `json:"accounts_reduce_only"`
	AccountsLiquidating int64           `json:"accounts_liquidating"`
	PendingHedges       int64           `json:"pending_hedges"`
	BufferedHedges      int64           `json:"buffered_hedges"`
	RetryingHedges      int64           `json:"retrying_hedges"`
	FailedHedges        int64           `json:"failed_hedges"`
	RecentLiquidations  int64           `json:"recent_liquidations"`
	UnhealthySymbols    int64           `json:"unhealthy_symbols"`
	TotalAbsoluteDrift  decimal.Decimal `json:"total_absolute_drift"`
	LastSnapshotAt      string          `json:"last_snapshot_at"`
}

type AdminHedgeTaskItem struct {
	ID                   uint64          `json:"id"`
	Symbol               string          `json:"symbol"`
	TriggerType          string          `json:"trigger_type"`
	InternalNetPosition  decimal.Decimal `json:"internal_net_position"`
	TargetHedgePosition  decimal.Decimal `json:"target_hedge_position"`
	CurrentHedgePosition decimal.Decimal `json:"current_hedge_position"`
	Drift                decimal.Decimal `json:"drift"`
	Status               string          `json:"status"`
	ErrorMessage         string          `json:"error_message"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
	LastOrderStatus      string          `json:"last_order_status"`
	LastOrderSide        string          `json:"last_order_side"`
	LastOrderSize        decimal.Decimal `json:"last_order_size"`
	LastOrderPrice       decimal.Decimal `json:"last_order_price"`
	LastOrderRetryCount  uint32          `json:"last_order_retry_count"`
}

type AdminRiskSnapshotItem struct {
	ID                    uint64          `json:"id"`
	Symbol                string          `json:"symbol"`
	TotalLongPosition     decimal.Decimal `json:"total_long_position"`
	TotalShortPosition    decimal.Decimal `json:"total_short_position"`
	NetPosition           decimal.Decimal `json:"net_position"`
	ExternalHedgePosition decimal.Decimal `json:"external_hedge_position"`
	Drift                 decimal.Decimal `json:"drift"`
	HedgeHealthy          bool            `json:"hedge_healthy"`
	TotalOpenInterest     decimal.Decimal `json:"total_open_interest"`
	CreatedAt             string          `json:"created_at"`
}

type AdminLiquidationItem struct {
	ID               uint64          `json:"id"`
	UserID           uint64          `json:"user_id"`
	Symbol           string          `json:"symbol"`
	Side             string          `json:"side"`
	Size             decimal.Decimal `json:"size"`
	EntryPrice       decimal.Decimal `json:"entry_price"`
	MarkPrice        decimal.Decimal `json:"mark_price"`
	LiquidationPrice decimal.Decimal `json:"liquidation_price"`
	ExecutionPrice   decimal.Decimal `json:"execution_price"`
	RealizedPnL      decimal.Decimal `json:"realized_pnl"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	CreatedAt        string          `json:"created_at"`
}

type AdminAlertItem struct {
	Level     string `json:"level"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Symbol    string `json:"symbol,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (s *AdminService) GetOverview() (*AdminOverview, error) {
	var overview AdminOverview

	if err := s.db.Model(&model.Symbol{}).Where("status = ?", "trading").Count(&overview.TradingSymbols).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Position{}).Where("status = ?", "open").Count(&overview.OpenPositions).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("status = ?", "at_risk").Count(&overview.AccountsAtRisk).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("status = ?", "reduce_only").Count(&overview.AccountsReduceOnly).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.User{}).Where("status = ?", "liquidating").Count(&overview.AccountsLiquidating).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.HedgeTask{}).Where("status = ?", "pending").Count(&overview.PendingHedges).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.HedgeTask{}).Where("status = ?", "buffered").Count(&overview.BufferedHedges).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.HedgeTask{}).Where("status = ?", "retrying").Count(&overview.RetryingHedges).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.HedgeTask{}).Where("status = ?", "failed").Count(&overview.FailedHedges).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&model.Liquidation{}).Where("created_at >= ?", time.Now().Add(-24*time.Hour)).Count(&overview.RecentLiquidations).Error; err != nil {
		return nil, err
	}

	var latestBySymbol []model.SystemRiskSnapshot
	if err := s.db.Raw(`
		SELECT s.*
		FROM system_risk_snapshots s
		INNER JOIN (
			SELECT symbol, MAX(id) AS max_id
			FROM system_risk_snapshots
			GROUP BY symbol
		) latest ON latest.max_id = s.id
	`).Scan(&latestBySymbol).Error; err != nil {
		return nil, err
	}
	totalAbsDrift := decimal.Zero
	var lastSnapshot time.Time
	for _, item := range latestBySymbol {
		if !item.HedgeHealthy {
			overview.UnhealthySymbols++
		}
		totalAbsDrift = totalAbsDrift.Add(item.Drift.Abs())
		if item.CreatedAt.After(lastSnapshot) {
			lastSnapshot = item.CreatedAt
		}
	}
	overview.TotalAbsoluteDrift = totalAbsDrift.Round(18)
	if !lastSnapshot.IsZero() {
		overview.LastSnapshotAt = lastSnapshot.UTC().Format(time.RFC3339)
	}

	return &overview, nil
}

func (s *AdminService) ListHedgeTasks(limit int) ([]AdminHedgeTaskItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var tasks []model.HedgeTask
	if err := s.db.Order("id desc").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	items := make([]AdminHedgeTaskItem, 0, len(tasks))
	for _, task := range tasks {
		var order model.HedgeOrder
		_ = s.db.Where("hedge_task_id = ?", task.ID).Order("id desc").First(&order).Error
		items = append(items, AdminHedgeTaskItem{
			ID:                   task.ID,
			Symbol:               task.Symbol,
			TriggerType:          task.TriggerType,
			InternalNetPosition:  task.InternalNetPosition,
			TargetHedgePosition:  task.TargetHedgePosition,
			CurrentHedgePosition: task.CurrentHedgePosition,
			Drift:                task.Drift,
			Status:               task.Status,
			ErrorMessage:         task.ErrorMessage,
			CreatedAt:            task.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:            task.UpdatedAt.UTC().Format(time.RFC3339),
			LastOrderStatus:      order.Status,
			LastOrderSide:        order.Side,
			LastOrderSize:        order.Size,
			LastOrderPrice:       order.Price,
			LastOrderRetryCount:  order.RetryCount,
		})
	}
	return items, nil
}

func (s *AdminService) ListRiskSnapshots(limit int) ([]AdminRiskSnapshotItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var snapshots []model.SystemRiskSnapshot
	if err := s.db.Order("id desc").Limit(limit).Find(&snapshots).Error; err != nil {
		return nil, err
	}
	items := make([]AdminRiskSnapshotItem, 0, len(snapshots))
	for _, item := range snapshots {
		items = append(items, AdminRiskSnapshotItem{
			ID:                    item.ID,
			Symbol:                item.Symbol,
			TotalLongPosition:     item.TotalLongPosition,
			TotalShortPosition:    item.TotalShortPosition,
			NetPosition:           item.NetPosition,
			ExternalHedgePosition: item.ExternalHedgePosition,
			Drift:                 item.Drift,
			HedgeHealthy:          item.HedgeHealthy,
			TotalOpenInterest:     item.TotalOpenInterest,
			CreatedAt:             item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *AdminService) ListLiquidations(limit int) ([]AdminLiquidationItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var liquidations []model.Liquidation
	if err := s.db.Order("id desc").Limit(limit).Find(&liquidations).Error; err != nil {
		return nil, err
	}
	items := make([]AdminLiquidationItem, 0, len(liquidations))
	for _, item := range liquidations {
		items = append(items, AdminLiquidationItem{
			ID:               item.ID,
			UserID:           item.UserID,
			Symbol:           item.Symbol,
			Side:             item.Side,
			Size:             item.Size,
			EntryPrice:       item.EntryPrice,
			MarkPrice:        item.MarkPrice,
			LiquidationPrice: item.LiquidationPrice,
			ExecutionPrice:   item.ExecutionPrice,
			RealizedPnL:      item.RealizedPnL,
			Type:             item.Type,
			Status:           item.Status,
			CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *AdminService) ListAlerts(limit int) ([]AdminAlertItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	alerts := make([]AdminAlertItem, 0, limit)

	var failedTasks []model.HedgeTask
	if err := s.db.Where("status IN ?", []string{"failed", "buffered", "retrying"}).Order("updated_at desc").Limit(limit).Find(&failedTasks).Error; err != nil {
		return nil, err
	}
	for _, task := range failedTasks {
		level := "warning"
		category := "hedge"
		title := "对冲待处理"
		if task.Status == "failed" {
			level = "error"
			title = "对冲失败"
		} else if task.Status == "retrying" {
			title = "对冲重试中"
		} else if task.Status == "buffered" {
			title = "对冲已缓冲"
		}
		alerts = append(alerts, AdminAlertItem{
			Level:     level,
			Category:  category,
			Title:     title,
			Detail:    task.ErrorMessage,
			Symbol:    task.Symbol,
			CreatedAt: task.UpdatedAt.UTC().Format(time.RFC3339),
		})
		if len(alerts) >= limit {
			return alerts[:limit], nil
		}
	}

	var snapshots []model.SystemRiskSnapshot
	if err := s.db.Where("hedge_healthy = ?", false).Order("created_at desc").Limit(limit).Find(&snapshots).Error; err != nil {
		return nil, err
	}
	for _, item := range snapshots {
		alerts = append(alerts, AdminAlertItem{
			Level:     "warning",
			Category:  "risk",
			Title:     "净敞口偏差超阈值",
			Detail:    "内部净敞口与外部对冲仓位偏差超过健康阈值，请检查 hedger 状态。",
			Symbol:    item.Symbol,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		})
		if len(alerts) >= limit {
			return alerts[:limit], nil
		}
	}

	var liquidations []model.Liquidation
	if err := s.db.Order("created_at desc").Limit(limit).Find(&liquidations).Error; err != nil {
		return nil, err
	}
	for _, item := range liquidations {
		alerts = append(alerts, AdminAlertItem{
			Level:     "error",
			Category:  "liquidation",
			Title:     "发生强平",
			Detail:    "检测到最近强平记录，请检查风险参数与用户仓位变化。",
			Symbol:    item.Symbol,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		})
		if len(alerts) >= limit {
			break
		}
	}

	return alerts, nil
}
