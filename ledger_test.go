package main

import (
	"errors"
	"strings"
	"testing"
)

func TestAddTransaction_BudgetLimit(t *testing.T) {
	l := NewLedger()
	l.SetBudget(Budget{Category: "еда", Limit: 1000.0})

	err := l.AddTransaction(Transaction{ID: 1, Category: "еда", Amount: 600.0})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	err = l.AddTransaction(Transaction{ID: 2, Category: "еда", Amount: 500.0})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected ErrBudgetExceeded, got %v", err)
	}

	txs := l.Transactions()
	if len(txs) != 1 {
		t.Fatalf("expected exactly 1 stored transaction, got %d", len(txs))
	}
}

func TestLoadBudgets_JSON(t *testing.T) {
	jsonContent := `[
		{"category": "книги", "limit": 1500.0},
		{"category": "спорт", "limit": 4000.0}
	]`

	l := NewLedger()
	if err := l.LoadBudgets(strings.NewReader(jsonContent)); err != nil {
		t.Fatalf("expected nil error on valid json, got %v", err)
	}

	b, ok := l.GetBudget("спорт")
	if !ok || b.Limit != 4000.0 {
		t.Fatalf("failed to retrieve updated budget for 'спорт'")
	}
}

func TestLoadBudgets_InvalidJSON(t *testing.T) {
	l := NewLedger()
	err := l.LoadBudgets(strings.NewReader(`invalid json string`))
	if err == nil {
		t.Fatal("expected error for malformed json, got nil")
	}
}