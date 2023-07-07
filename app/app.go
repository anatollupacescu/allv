package app

import (
	"context"
	"errors"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type App struct {
	c *ethclient.Client
}

func New(endpoint string) *App {
	if endpoint == "" {
		log.Fatal("RPC endpoint not provided")
	}
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		log.Fatalf("dial to endpoint '%s': %v", endpoint, err)
	}

	return &App{c: client}
}

var (
	ErrInvalidAddress = errors.New("address is not a valid hex format")
	ErrRPCCall        = errors.New("call remote service")
)

func (app *App) GetBalance(ctx context.Context, addrS string) (*big.Int, error) {
	if !common.IsHexAddress(addrS) {
		return nil, ErrInvalidAddress
	}

	account := common.HexToAddress(addrS)

	balance, err := app.c.BalanceAt(ctx, account, nil)
	if err != nil {
		return nil, errors.Join(ErrRPCCall, err)
	}

	return balance, nil
}
