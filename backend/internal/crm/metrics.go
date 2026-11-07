package crm

import (
	"context"
	"fmt"
)

// CrmMetrics — базовые показатели CRM. Суммы в тенге строками.
type CrmMetrics struct {
	ClientsTotal      int64  `json:"clientsTotal"`
	DealsActive       int64  `json:"dealsActive" doc:"Сделки не на этапах «Договор заключен» и «Отказ»"`
	DealsWon          int64  `json:"dealsWon"`
	DealsAmountActive string `json:"dealsAmountActive" example:"1250000.00"`
	DealsAmountWon    string `json:"dealsAmountWon" example:"980000.00"`
	TasksOverdue      int64  `json:"tasksOverdue"`
	TasksOpen         int64  `json:"tasksOpen"`
}

func (s *Service) Metrics(ctx context.Context) (CrmMetrics, error) {
	r, err := s.q.CrmMetrics(ctx, s.today())
	if err != nil {
		return CrmMetrics{}, fmt.Errorf("показатели CRM: %w", err)
	}
	return CrmMetrics{
		ClientsTotal: r.ClientsTotal, DealsActive: r.DealsActive, DealsWon: r.DealsWon,
		DealsAmountActive: r.DealsAmountActive.StringFixed(2), DealsAmountWon: r.DealsAmountWon.StringFixed(2),
		TasksOverdue: r.TasksOverdue, TasksOpen: r.TasksOpen,
	}, nil
}
