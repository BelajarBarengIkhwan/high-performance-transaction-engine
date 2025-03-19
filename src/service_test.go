package main

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	service *Service
)

func TestMain(m *testing.M) {
	logger := logrus.New()
	db := InitDatabase()
	rdb := InitRedis()
	Seeding(NUM_ACCOUNT, INITIAL_BALANCE, db, rdb, logger)
	service = NewService(db, rdb, logger)
	os.Exit(m.Run())
}

func WithdrawWrapper(wg *sync.WaitGroup, errChan chan<- error, acc string, amount decimal.Decimal, withdraw func(acc string, amount decimal.Decimal) error) {
	wg.Add(1)
	defer wg.Done()
	err := withdraw(acc, amount)
	errChan <- err
}

func WithdrawUnderBalance(t *testing.T, withdraw func(acc string, amount decimal.Decimal) error) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 0)
	testBalance := decimal.NewFromInt(999_999)
	err := withdraw(acc, testBalance)
	assert.NoError(t, err, ErrInsufficientBalance)
}

func WithdrawOverBalance(t *testing.T, withdraw func(acc string, amount decimal.Decimal) error) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 1)
	testBalance := decimal.NewFromInt(1_000_001)
	err := withdraw(acc, testBalance)
	assert.Error(t, err, ErrInsufficientBalance)
}

func ConcurrencyWithdraw(t *testing.T, withdraw func(acc string, amount decimal.Decimal) error) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 2)
	testBalance := decimal.NewFromInt(500_000)
	goroutineCount := 4
	wg := sync.WaitGroup{}
	errChan := make(chan error, goroutineCount)
	for range goroutineCount {
		go WithdrawWrapper(&wg, errChan, acc, testBalance, service.Withdraw)
	}
	wg.Wait()

	errCount := 0
	expectedErrCount := 2
	for range goroutineCount {
		err := <-errChan
		if err != nil {
			errCount += 1
		}
	}
	assert.Equal(t, expectedErrCount, errCount, "withdraw failed is not as expected")
}

func TestDatabaseWithdraw(t *testing.T) {
	t.Run("withdraw under balance", func(t *testing.T) { WithdrawUnderBalance(t, service.Withdraw) })
	t.Run("withdraw over balance", func(t *testing.T) { WithdrawOverBalance(t, service.Withdraw) })
	t.Run("concurrency withdraw", func(t *testing.T) { ConcurrencyWithdraw(t, service.Withdraw) })
}

func TestFastWithdraw(t *testing.T) {
	t.Run("fast withdraw under balance", func(t *testing.T) { WithdrawUnderBalance(t, service.FastWithdraw) })
	t.Run("fast withdraw over balance", func(t *testing.T) { WithdrawOverBalance(t, service.FastWithdraw) })
	t.Run("fast concurrency withdraw", func(t *testing.T) { ConcurrencyWithdraw(t, service.FastWithdraw) })
}
