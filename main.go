package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrInvalidAmount  = errors.New("invalid transaction amount")
)

type Transaction struct {
	ID       int     `json:"id"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Comment  string  `json:"comment,omitempty"`
}

type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

type Ledger struct {
	transactions []Transaction
	budgets      map[string]Budget
}

func NewLedger() *Ledger {
	l := &Ledger{
		transactions: make([]Transaction, 0),
		budgets:      make(map[string]Budget),
	}

	l.SetBudget(Budget{Category: "еда", Limit: 5000.0, Period: "месяц"})
	l.SetBudget(Budget{Category: "коммуналка", Limit: 7000.0, Period: "месяц"})

	return l
}

func (l *Ledger) SetBudget(b Budget) {
	if l.budgets == nil {
		l.budgets = make(map[string]Budget)
	}
	l.budgets[b.Category] = b
}

func (l *Ledger) GetBudget(category string) (Budget, bool) {
	b, ok := l.budgets[category]
	return b, ok
}

func (l *Ledger) CurrentSpent(category string) float64 {
	var total float64
	for _, tx := range l.transactions {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total
}

func (l *Ledger) AddTransaction(tx Transaction) error {
	if tx.Amount <= 0 {
		return fmt.Errorf("%w: сумма должна быть больше нуля, получено %.2f", ErrInvalidAmount, tx.Amount)
	}

	budget, exists := l.budgets[tx.Category]
	if exists {
		spent := l.CurrentSpent(tx.Category)
		if spent+tx.Amount > budget.Limit {
			return fmt.Errorf("%w: категория %q, лимит %.2f, уже потрачено %.2f, попытка списать %.2f",
				ErrBudgetExceeded, tx.Category, budget.Limit, spent, tx.Amount)
		}
	}

	l.transactions = append(l.transactions, tx)
	return nil
}

func (l *Ledger) LoadBudgets(r io.Reader) error {
	if r == nil {
		return errors.New("reader cannot be nil")
	}

	var loaded []Budget
	dec := json.NewDecoder(r)
	if err := dec.Decode(&loaded); err != nil {
		return fmt.Errorf("декодирование json: %w", err)
	}

	for _, b := range loaded {
		if b.Category == "" {
			return errors.New("категория бюджета не может быть пустой")
		}
		if b.Limit < 0 {
			return fmt.Errorf("лимит для категории %q не может быть отрицательным", b.Category)
		}
		l.SetBudget(b)
	}

	return nil
}

func (l *Ledger) Transactions() []Transaction {
	res := make([]Transaction, len(l.transactions))
	copy(res, l.transactions)
	return res
}

func main() {
	ledger := NewLedger()

	fmt.Println("=== 1. Проверка обработки ошибок I/O ===")
	if _, err := os.Open("nonexistent_budgets.json"); err != nil {
		fmt.Printf("Успешный перехват ошибки открытия: %v\n\n", err)
	}

	fmt.Println("=== 2. Загрузка бюджетов из файла JSON ===")
	file, err := os.Open("budgets.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "критическая ошибка открытия budgets.json: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	if err := ledger.LoadBudgets(reader); err != nil {
		fmt.Fprintf(os.Stderr, "критическая ошибка парсинга бюджетов: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Бюджеты успешно импортированы через bufio.NewReader")

	fmt.Println("\n=== 3. Добавление и изменение бюджета через SetBudget ===")
	ledger.SetBudget(Budget{Category: "развлечения", Limit: 1500.0, Period: "месяц"})
	fmt.Println("Установлен лимит на категорию 'развлечения': 1500.00")

	fmt.Println("\n=== 4. Выполнение транзакций в пределах лимита ===")
	tx1 := Transaction{ID: 1, Category: "еда", Amount: 2000.0, Comment: "Супермаркет"}
	if err := ledger.AddTransaction(tx1); err != nil {
		fmt.Printf("Ошибка tx1: %v\n", err)
	} else {
		fmt.Printf("OK: транзакция #%d принята. Категория %q: %.2f (израсходовано: %.2f)\n",
			tx1.ID, tx1.Category, tx1.Amount, ledger.CurrentSpent(tx1.Category))
	}

	tx2 := Transaction{ID: 2, Category: "еда", Amount: 2500.0, Comment: "Продуктовый рынок"}
	if err := ledger.AddTransaction(tx2); err != nil {
		fmt.Printf("Ошибка tx2: %v\n", err)
	} else {
		fmt.Printf("OK: транзакция #%d принята. Категория %q: %.2f (израсходовано: %.2f)\n",
			tx2.ID, tx2.Category, tx2.Amount, ledger.CurrentSpent(tx2.Category))
	}

	fmt.Println("\n=== 5. Попытка превышения бюджета ===")
	tx3 := Transaction{ID: 3, Category: "еда", Amount: 1000.0, Comment: "Ресторан"}
	if err := ledger.AddTransaction(tx3); err != nil {
		if errors.Is(err, ErrBudgetExceeded) {
			fmt.Printf("ОТКАЗ (ожидаемо): %v\n", err)
		} else {
			fmt.Printf("Непредвиденная ошибка: %v\n", err)
		}
	} else {
		fmt.Println("ОШИБКА: транзакция с превышением была ошибочно сохранена!")
	}

	fmt.Println("\n=== 6. Транзакция по категории без установленного лимита ===")
	tx4 := Transaction{ID: 4, Category: "аптека", Amount: 850.0, Comment: "Лекарства"}
	if err := ledger.AddTransaction(tx4); err != nil {
		fmt.Printf("Ошибка tx4: %v\n", err)
	} else {
		fmt.Printf("OK: транзакция без лимита #%d принята. Категория %q: %.2f\n",
			tx4.ID, tx4.Category, tx4.Amount)
	}

	fmt.Println("\n=== 7. Итоговый список транзакций в сервисе ===")
	saved := ledger.Transactions()
	for _, tx := range saved {
		fmt.Printf("- ID: %d | %-12s | Сумма: %7.2f руб. | %s\n",
			tx.ID, tx.Category, tx.Amount, tx.Comment)
	}
	fmt.Printf("\nИтого сохранено транзакций: %d (отклоненная транзакция #3 отсутствует)\n", len(saved))
}