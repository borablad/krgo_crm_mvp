package crm

import (
	"context"
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/borablad/krgo_crm_mvp/backend/internal/audit"
	"github.com/borablad/krgo_crm_mvp/backend/internal/platform/pg"
)

func newSvc(t *testing.T) (*Service, *pgxpool.Pool, int64) {
	t.Helper()
	pool := pg.TestPool(t)
	var uid int64
	err := pool.QueryRow(context.Background(),
		`insert into users (login, full_name, role, password_hash) values ('manager', 'Менеджер', 'employee', 'x') returning id`).Scan(&uid)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(pool, audit.NewRecorder()), pool, uid
}

// status и fieldErrors разбирают готовую HTTP-ошибку Huma.
func status(err error) int {
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		return m.Status
	}
	return 0
}

func fieldErrors(err error) map[string]string {
	out := map[string]string{}
	var m *huma.ErrorModel
	if errors.As(err, &m) {
		for _, d := range m.Errors {
			out[d.Location] = d.Message
		}
	}
	return out
}

func mustClient(t *testing.T, s *Service, owner int64, name string) CrmClient {
	t.Helper()
	c, err := s.CreateClient(context.Background(), ClientInput{Name: name, OwnerUserID: owner})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestClientValidation(t *testing.T) {
	s, _, uid := newSvc(t)
	ctx := context.Background()

	_, err := s.CreateClient(ctx, ClientInput{Name: "ТОО Тест", Bin: "12345", OwnerUserID: uid})
	if fe := fieldErrors(err); status(err) != 422 || fe["body.bin"] != "Введите 12 цифр БИН/ИИН" {
		t.Fatalf("короткий БИН: %v %v", err, fe)
	}
	_, err = s.CreateClient(ctx, ClientInput{Name: "ТОО Тест", Bin: "123456789012", OwnerUserID: uid})
	if fe := fieldErrors(err); fe["body.bin"] != "Неверная контрольная сумма БИН/ИИН" {
		t.Fatalf("контрольная сумма: %v", fe)
	}
	_, err = s.CreateClient(ctx, ClientInput{Name: " ", Email: "no-at", OwnerUserID: 999})
	fe := fieldErrors(err)
	if fe["body.name"] == "" || fe["body.email"] == "" || fe["body.ownerUserId"] != "Выберите ответственного" {
		t.Fatalf("пустое имя, почта, ответственный: %v", fe)
	}
	c, err := s.CreateClient(ctx, ClientInput{Name: "ТОО Тест", Bin: "150340410002", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != "potential" || c.OwnerName != "Менеджер" || c.Bin != "150340410002" {
		t.Fatalf("клиент: %+v", c)
	}
	list, total, err := s.ListClients(ctx, ClientFilter{Q: "1503404"})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("поиск по БИН: %v %d", err, total)
	}
}

func TestDealValidationAndStages(t *testing.T) {
	s, pool, uid := newSvc(t)
	ctx := context.Background()
	cl := mustClient(t, s, uid, "Клиент")

	_, err := s.CreateDeal(ctx, DealInput{ClientID: cl.ID, Title: "Сделка", Amount: "-5", Probability: 101, OwnerUserID: uid})
	fe := fieldErrors(err)
	if fe["body.amount"] == "" || fe["body.probability"] == "" {
		t.Fatalf("ожидались ошибки суммы и вероятности: %v", fe)
	}
	_, err = s.CreateDeal(ctx, DealInput{ClientID: cl.ID, Title: "Сделка", Amount: "abc", Probability: 50, OwnerUserID: uid})
	if fieldErrors(err)["body.amount"] != "Введите сумму числом, например 125000.00" {
		t.Fatalf("сумма-текст: %v", err)
	}
	_, err = s.CreateDeal(ctx, DealInput{ClientID: 999, Title: "Сделка", Amount: "1", Probability: 50, OwnerUserID: uid})
	if fieldErrors(err)["body.clientId"] == "" {
		t.Fatalf("несуществующий клиент: %v", err)
	}

	d, err := s.CreateDeal(ctx, DealInput{ClientID: cl.ID, Title: "Сделка", Amount: "125000.00", Probability: 50, OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	if d.Stage != "new" || d.Amount != "125000.00" || d.ClientName != "Клиент" || len(d.History) != 1 {
		t.Fatalf("новая сделка: %+v", d)
	}
	changed := d.StageChangedAt

	d, err = s.ChangeStage(ctx, d.ID, "negotiation", "созвонились")
	if err != nil {
		t.Fatal(err)
	}
	if d.Stage != "negotiation" || !d.StageChangedAt.After(changed) || len(d.History) != 2 {
		t.Fatalf("после смены этапа: %+v", d)
	}
	h := d.History[1]
	if h.FromStage != "new" || h.ToStage != "negotiation" || h.Comment != "созвонились" {
		t.Fatalf("история: %+v", h)
	}
	var n int
	if err := pool.QueryRow(ctx, `select count(*) from audit_log where action = 'deal.stage'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("запись журнала о смене этапа: %d %v", n, err)
	}

	if _, err = s.ChangeStage(ctx, d.ID, "negotiation", ""); status(err) != 422 {
		t.Fatalf("тот же этап: %v", err)
	}
	if _, err = s.ChangeStage(ctx, d.ID, "won", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ChangeStage(ctx, d.ID, "proposal", ""); status(err) != 422 {
		t.Fatalf("после won: %v", err)
	}
}

func TestTasks(t *testing.T) {
	s, _, uid := newSvc(t)
	ctx := context.Background()
	cl := mustClient(t, s, uid, "Клиент")
	deal, err := s.CreateDeal(ctx, DealInput{ClientID: cl.ID, Title: "Сделка", Amount: "100", Probability: 50, OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	today := s.today()

	if _, err := s.CreateTask(ctx, TaskInput{AssigneeUserID: uid}); len(fieldErrors(err)) < 2 {
		t.Fatalf("пустая задача: %v", err)
	}
	mk := func(title string, offset int, dealID int64) CrmTask {
		task, err := s.CreateTask(ctx, TaskInput{Title: title, AssigneeUserID: uid, DueDate: today.AddDate(0, 0, offset), DealID: dealID})
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	past := mk("вчера", -1, deal.ID)
	now := mk("сегодня", 0, 0)
	fut := mk("завтра", 1, 0)

	if !past.IsOverdue || now.IsOverdue || fut.IsOverdue {
		t.Fatalf("просрочка: %v %v %v", past.IsOverdue, now.IsOverdue, fut.IsOverdue)
	}
	if past.ClientID != cl.ID || past.DealTitle != "Сделка" {
		t.Fatalf("клиент должен подставиться из сделки: %+v", past)
	}
	overdue, total, err := s.ListTasks(ctx, TaskFilter{Overdue: true})
	if err != nil || total != 1 || overdue[0].ID != past.ID {
		t.Fatalf("фильтр просроченных: %v %d", err, total)
	}

	done, err := s.CompleteTask(ctx, past.ID)
	if err != nil || done.Status != "done" || done.DoneAt == nil || done.IsOverdue {
		t.Fatalf("выполнение: %+v %v", done, err)
	}
	if _, err := s.CompleteTask(ctx, past.ID); status(err) != 422 {
		t.Fatalf("повторное выполнение: %v", err)
	}
	cancelled, err := s.CancelTask(ctx, fut.ID)
	if err != nil || cancelled.Status != "cancelled" {
		t.Fatalf("отмена: %+v %v", cancelled, err)
	}
}

func TestMetrics(t *testing.T) {
	s, _, uid := newSvc(t)
	ctx := context.Background()
	cl := mustClient(t, s, uid, "Клиент 1")
	mustClient(t, s, uid, "Клиент 2")

	mkDeal := func(amount string) CrmDealDetail {
		d, err := s.CreateDeal(ctx, DealInput{ClientID: cl.ID, Title: "Сделка", Amount: amount, Probability: 50, OwnerUserID: uid})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	mkDeal("1000.50")
	mkDeal("2000.00")
	won := mkDeal("500.25")
	lost := mkDeal("700.00")
	if _, err := s.ChangeStage(ctx, won.ID, "won", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeStage(ctx, lost.ID, "lost", ""); err != nil {
		t.Fatal(err)
	}
	today := s.today()
	for _, off := range []int{-3, -1, 2} {
		if _, err := s.CreateTask(ctx, TaskInput{Title: "задача", AssigneeUserID: uid, DueDate: today.AddDate(0, 0, off)}); err != nil {
			t.Fatal(err)
		}
	}

	m, err := s.Metrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := CrmMetrics{
		ClientsTotal: 2, DealsActive: 2, DealsWon: 1, DealsAmountActive: "3000.50", DealsAmountWon: "500.25",
		TasksOverdue: 2, TasksOpen: 3,
	}
	if m != want {
		t.Fatalf("показатели:\n got %+v\nwant %+v", m, want)
	}
}

func TestSeedDemoIsIdempotent(t *testing.T) {
	s, pool, uid := newSvc(t)
	ctx := context.Background()
	var director int64
	if err := pool.QueryRow(ctx, `insert into users (login, full_name, role, password_hash) values ('director', 'Директор', 'manager', 'x') returning id`).Scan(&director); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{"manager": uid, "director": director}
	if err := SeedDemo(ctx, s, ids); err != nil {
		t.Fatal(err)
	}
	if err := SeedDemo(ctx, s, ids); err != nil {
		t.Fatal(err)
	}
	m, err := s.Metrics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if m.ClientsTotal != 8 || m.DealsWon != 2 || m.TasksOverdue != 3 || m.TasksOpen != 6 {
		t.Fatalf("демо-данные: %+v", m)
	}
}

// Регрессии из ревью: сделка чужого клиента в задаче и сумма с тремя знаками после запятой.
func TestReviewRegressions(t *testing.T) {
	s, _, uid := newSvc(t)
	ctx := context.Background()
	c1 := mustClient(t, s, uid, "Первый")
	c2 := mustClient(t, s, uid, "Второй")
	d2, err := s.CreateDeal(ctx, DealInput{ClientID: c2.ID, Title: "Сделка второго", Amount: "100", Probability: 50, OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateTask(ctx, TaskInput{Title: "Позвонить", AssigneeUserID: uid, DueDate: s.today(), ClientID: c1.ID, DealID: d2.ID})
	if _, ok := fieldErrors(err)["body.dealId"]; !ok {
		t.Fatalf("сделка другого клиента должна давать ошибку у dealId: %v", err)
	}

	_, err = s.CreateDeal(ctx, DealInput{ClientID: c1.ID, Title: "Копейки", Amount: "10.345", Probability: 50, OwnerUserID: uid})
	if _, ok := fieldErrors(err)["body.amount"]; !ok {
		t.Fatalf("три знака после запятой должны давать ошибку у amount: %v", err)
	}
	d, err := s.CreateDeal(ctx, DealInput{ClientID: c1.ID, Title: "Ровно", Amount: "10.340", Probability: 50, OwnerUserID: uid})
	if err != nil || d.Amount != "10.34" {
		t.Fatalf("10.340 должно сохраниться как 10.34: %q %v", d.Amount, err)
	}
}
