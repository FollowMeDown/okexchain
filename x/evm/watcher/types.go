package watcher

import (
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	rpctypes "github.com/okex/okexchain/app/rpc/types"
	"github.com/okex/okexchain/x/evm/types"
	"github.com/status-im/keycard-go/hexutils"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	prefixTx           = "0x1"
	prefixBlock        = "0x2"
	prefixReceipt      = "0x3"
	prefixCode         = "0x4"
	prefixBlockInfo    = "0x5"
	prefixLatestHeight = "0x6"

	KeyLatestHeight = "LatestHeight"
)

type WatchMessage interface {
	GetKey() string
	GetValue() string
}

type MsgEthTx struct {
	Key       string
	JsonEthTx string
}

func NewMsgEthTx(tx *types.MsgEthereumTx, txHash, blockHash common.Hash, height, index uint64) *MsgEthTx {
	ethTx, e := rpctypes.NewTransaction(tx, txHash, blockHash, height, index)
	if e != nil {
		return nil
	}
	jsTx, e := json.Marshal(ethTx)
	if e != nil {
		return nil
	}
	msg := MsgEthTx{
		Key:       hexutils.BytesToHex(txHash.Bytes()),
		JsonEthTx: string(jsTx),
	}
	return &msg
}

func (m MsgEthTx) GetKey() string {
	return prefixTx + m.Key
}

func (m MsgEthTx) GetValue() string {
	return m.JsonEthTx
}

type MsgCode struct {
	Key  string
	Code string
}

type CodeInfo struct {
	Height uint64 `height`
	Code   string `code`
}

func NewMsgCode(contractAddr common.Address, code []byte, height uint64) *MsgCode {
	codeInfo := CodeInfo{
		Height: height,
		Code:   hexutils.BytesToHex(code),
	}
	jsCode, e := json.Marshal(codeInfo)
	if e != nil {
		return nil
	}
	return &MsgCode{
		Key:  contractAddr.String(),
		Code: string(jsCode),
	}
}

func (m MsgCode) GetKey() string {
	return prefixCode + m.Key
}

func (m MsgCode) GetValue() string {
	return m.Code
}

type MsgTransactionReceipt struct {
	txHash  string
	receipt string
}

type TransactionReceipt struct {
	Status            uint32          `json:"status"`
	CumulativeGasUsed uint64          `json:"cumulativeGasUsed"`
	LogsBloom         ethtypes.Bloom  `json:"logsBloom"`
	Logs              []*ethtypes.Log `json:"logs"`
	TransactionHash   string          `json:"transactionHash"`
	ContractAddress   string          `json:"contractAddress"`
	GasUsed           uint64          `json:"gasUsed"`
	BlockHash         string          `json:"blockHash"`
	BlockNumber       uint64          `json:"blockNumber"`
	TransactionIndex  uint64          `json:"transactionIndex"`
	From              string          `json:"from"`
	To                string          `json:"to"`
}

func NewMsgTransactionReceipt(status uint32, tx *types.MsgEthereumTx, txHash, blockHash common.Hash, txIndex, height uint64, data *types.ResultData, cumulativeGas, GasUsed uint64) *MsgTransactionReceipt {
	toAddr := ""
	if tx.To() != nil {
		toAddr = tx.To().String()
	}
	tr := TransactionReceipt{
		Status:            status,
		CumulativeGasUsed: cumulativeGas,
		LogsBloom:         data.Bloom,
		Logs:              data.Logs,
		TransactionHash:   txHash.String(),
		ContractAddress:   data.ContractAddress.String(),
		GasUsed:           GasUsed,
		BlockHash:         blockHash.String(),
		BlockNumber:       height,
		TransactionIndex:  txIndex,
		From:              tx.From().String(),
		To:                toAddr,
	}
	jsTr, e := json.Marshal(tr)
	if e != nil {
		return nil
	}
	return &MsgTransactionReceipt{txHash: txHash.String(), receipt: string(jsTr)}
}

func (m MsgTransactionReceipt) GetKey() string {
	return prefixReceipt + m.txHash
}

func (m MsgTransactionReceipt) GetValue() string {
	return m.receipt
}

type MsgBlock struct {
	blockHash string
	block     string
}

type EthBlock struct {
	Number           uint64         `json:"number"`
	Hash             common.Hash    `json:"hash"`
	ParentHash       common.Hash    `json:"parentHash"`
	Nonce            uint64         `json:"nonce"`
	Sha3Uncles       common.Hash    `json:"sha3Uncles"`
	LogsBloom        ethtypes.Bloom `json:"logsBloom"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	StateRoot        common.Hash    `json:"stateRoot"`
	Miner            common.Address `json:"miner"`
	MixHash          common.Hash    `json:"mixHash"`
	Difficulty       uint64         `json:"difficulty"`
	TotalDifficulty  uint64         `json:"totalDifficulty"`
	ExtraData        hexutil.Bytes  `json:"extraData"`
	Size             uint64         `json:"size"`
	GasLimit         uint64         `json:"gasLimit"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        uint64         `json:"timestamp"`
	Uncles           []string       `json:"uncles"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	Transactions     interface{}    `json:"transactions"`
}

func NewMsgBlock(height uint64, blockBloom ethtypes.Bloom, blockHash common.Hash, header abci.Header, gasLimit uint64, gasUsed *big.Int, txs interface{}) *MsgBlock {
	b := EthBlock{
		Number:           height,
		Hash:             blockHash,
		ParentHash:       common.BytesToHash(header.LastBlockId.Hash),
		Nonce:            0,
		Sha3Uncles:       common.Hash{},
		LogsBloom:        blockBloom,
		TransactionsRoot: common.BytesToHash(header.DataHash),
		StateRoot:        common.BytesToHash(header.AppHash),
		Miner:            common.Address{},
		MixHash:          common.Hash{},
		Difficulty:       0,
		TotalDifficulty:  0,
		ExtraData:        nil,
		Size:             0,
		GasLimit:         gasLimit,
		GasUsed:          gasUsed,
		Timestamp:        uint64(header.Time.Unix()),
		Uncles:           []string{},
		ReceiptsRoot:     common.Hash{},
		Transactions:     txs,
	}
	jsBlock, e := json.Marshal(b)
	if e != nil {
		return nil
	}
	return &MsgBlock{blockHash: blockHash.String(), block: string(jsBlock)}
}

func (m MsgBlock) GetKey() string {
	return prefixBlock + m.blockHash
}

func (m MsgBlock) GetValue() string {
	return m.block
}

type MsgBlockInfo struct {
	height string
	hash   string
}

func NewMsgBlockInfo(height uint64, blockHash common.Hash) *MsgBlockInfo {
	return &MsgBlockInfo{
		height: strconv.Itoa(int(height)),
		hash:   blockHash.String(),
	}
}

func (b MsgBlockInfo) GetKey() string {
	return prefixBlockInfo + b.height
}

func (b MsgBlockInfo) GetValue() string {
	return b.hash
}

type MsgLatestHeight struct {
	height string
}

func NewMsgLatestHeight(height uint64) *MsgLatestHeight {
	return &MsgLatestHeight{
		height: strconv.Itoa(int(height)),
	}
}

func (b MsgLatestHeight) GetKey() string {
	return prefixLatestHeight + KeyLatestHeight
}

func (b MsgLatestHeight) GetValue() string {
	return b.height
}
