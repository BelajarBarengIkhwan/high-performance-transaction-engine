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

func WithdrawWrapper(wg *sync.WaitGroup, errChan chan<- error, acc string, amount decimal.Decimal) {
	wg.Add(1)
	defer wg.Done()
	err := service.Withdraw(acc, amount)
	errChan <- err
}

func TestWithdrawUnderBalance(t *testing.T) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 0)
	testBalance := decimal.NewFromInt(999_999)
	err := service.Withdraw(acc, testBalance.Add(decimal.NewFromInt(1)))
	assert.NoError(t, err, ErrInsufficientBalance)
}

func TestWithdrawOverBalance(t *testing.T) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 1)
	testBalance := decimal.NewFromInt(1_000_001)
	err := service.Withdraw(acc, testBalance.Mul(decimal.NewFromInt(2)))
	assert.Error(t, err, ErrInsufficientBalance)
}

func TestConcurrencyWithdraw(t *testing.T) {
	acc := fmt.Sprintf(ACCOUNT_PREFIX, 2)
	testBalance := decimal.NewFromInt(500_000)
	goroutineCount := 4
	wg := sync.WaitGroup{}
	errChan := make(chan error, goroutineCount)
	for range goroutineCount {
		go WithdrawWrapper(&wg, errChan, acc, testBalance)
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

func TestWithdraw(t *testing.T) {
	t.Run("withdraw under balance", TestWithdrawUnderBalance)
	t.Run("withdraw over balance", TestWithdrawOverBalance)
	t.Run("concurrency withdraw", TestConcurrencyWithdraw)
}
