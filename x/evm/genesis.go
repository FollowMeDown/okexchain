package evm

import (
	"fmt"
	"io/ioutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	ethcmn "github.com/ethereum/go-ethereum/common"
	ethermint "github.com/okex/okexchain/app/types"
	"github.com/okex/okexchain/x/evm/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	absolutePath           = "/tmp/okexchain"
	absoluteCodePath       = absolutePath + "/code/"
	absoluteStoragePath    = absolutePath + "/storage/"
	absoluteTxlogsFilePath = absolutePath + "/txlogs/"

	codeFileSuffix    = ".code"
	storageFileSuffix = ".storage"
	txlogsFileSuffix  = ".json"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(ctx sdk.Context, k Keeper, accountKeeper types.AccountKeeper, data GenesisState) []abci.ValidatorUpdate { // nolint: interfacer
	k.SetParams(ctx, data.Params)

	evmDenom := data.Params.EvmDenom

	for _, account := range data.Accounts {
		address := ethcmn.HexToAddress(account.Address)
		accAddress := sdk.AccAddress(address.Bytes())

		// check that the EVM balance the matches the account balance
		acc := accountKeeper.GetAccount(ctx, accAddress)
		if acc == nil {
			panic(fmt.Errorf("account not found for address %s", account.Address))
		}

		_, ok := acc.(*ethermint.EthAccount)
		if !ok {
			panic(
				fmt.Errorf("account %s must be an %T type, got %T",
					account.Address, &ethermint.EthAccount{}, acc,
				),
			)
		}

		evmBalance := acc.GetCoins().AmountOf(evmDenom)
		k.SetNonce(ctx, address, acc.GetSequence())
		k.SetBalance(ctx, address, evmBalance.BigInt())

		// read Code from file
		codeFilePath := absoluteCodePath + account.Address + codeFileSuffix
		if pathExist(codeFilePath) {
			code := readCodeFromFile(codeFilePath)
			k.SetCodeDirectly(ctx, code)
		}

		// read Storage From file
		storageFilePath := absoluteStoragePath + account.Address + storageFileSuffix
		if pathExist(storageFilePath) {
			storage := readStorageFromFile(storageFilePath)
			for _, state := range storage {
				k.SetStateDirectly(ctx, address, state.Key, state.Value)
			}
		}
	}

	if pathExist(absoluteTxlogsFilePath) {
		fileInfos, err := ioutil.ReadDir(absoluteTxlogsFilePath)
		if err != nil {
			panic(err)
		}

		for _, fileInfo := range fileInfos {
			hash := convertHexStrToHash(fileInfo.Name())
			logs := readTxLogsFromFile(absoluteTxlogsFilePath + fileInfo.Name())
			k.SetTxLogsDirectly(ctx, hash, logs)
		}
	}

	k.SetChainConfig(ctx, data.ChainConfig)

	// set state objects and code to store
	_, err := k.Commit(ctx, false)
	if err != nil {
		panic(err)
	}

	// set storage to store
	// NOTE: don't delete empty object to prevent import-export simulation failure
	err = k.Finalise(ctx, false)
	if err != nil {
		panic(err)
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k Keeper, ak types.AccountKeeper) GenesisState {
	initExportEnv()

	// nolint: prealloc
	var ethGenAccounts []types.GenesisAccount
	ak.IterateAccounts(ctx, func(account authexported.Account) bool {
		ethAccount, ok := account.(*ethermint.EthAccount)
		if !ok {
			// ignore non EthAccounts
			return false
		}

		addr := ethAccount.EthAddress()

		// write Code
		go syncWriteAccountCode(ctx, k, addr)
		// write Storage
		go syncWriteAccountStorageSlice(ctx, k, addr)

		genAccount := types.GenesisAccount{
			Address: addr.String(),
			Code:    nil,
			Storage: nil,
		}

		ethGenAccounts = append(ethGenAccounts, genAccount)
		return false
	})

	// write tx logs
	writeAllTxLogs(ctx, k)

	// wait for all data to be written into files
	globalWG.Wait()

	config, _ := k.GetChainConfig(ctx)

	return GenesisState{
		Accounts:    ethGenAccounts,
		TxsLogs:     []types.TransactionLogs{}, //todo
		ChainConfig: config,
		Params:      k.GetParams(ctx),
	}
}
