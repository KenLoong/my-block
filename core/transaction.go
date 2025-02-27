package core

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"warson-blockchain/crypto"
	"warson-blockchain/types"
)

type TxType byte

const (
	TxTypeCollection TxType = iota // 0x0
	TxTypeMint                     // 0x01
)

type CollectionTx struct {
	Fee      int64
	MetaData []byte
}

type MintTx struct {
	Fee             int64
	NFT             types.Hash
	Collection      types.Hash
	MetaData        []byte
	CollectionOwner crypto.PublicKey
	Signature       crypto.Signature
}

type Transaction struct {
	// Any arbitrary data for the VM
	Data  []byte
	To    crypto.PublicKey
	Value uint64
	// Only used for native NFT logic
	TxInner   any `json:"tx_inner,omitempty"`
	From      crypto.PublicKey
	Signature *crypto.Signature `json:"signature,omitempty"`
	Nonce     int64
	TXHash    types.Hash // cached version of the tx data hash
}

// NewTransaction creates a new transaction with random nonce
func NewTransaction(data []byte) *Transaction {
	return &Transaction{
		Data:  data,
		Nonce: rand.Int63n(10000000000000),
	}
}

// MarshalJSON provides custom encoding for Transaction
/*func (tx *Transaction) MarshalJSON() ([]byte, error) {

	type Alias Transaction
	aux := &struct {
		TxInner json.RawMessage `json:"tx_inner"`
		*Alias
	}{
		Alias: (*Alias)(tx),
	}

	return json.Marshal(aux)
}
*/

// UnmarshalJSON provides custom decoding for Transaction
func (tx *Transaction) UnmarshalJSON(data []byte) error {
	type Alias Transaction
	aux := &struct {
		TxInner json.RawMessage `json:"tx_inner"`
		*Alias
	}{
		Alias: (*Alias)(tx),
	}

	// 先解码常规字段
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// 如果 TxInner 为空，直接返回
	if len(aux.TxInner) == 0 {
		return nil
	}

	// 尝试逐步匹配类型
	// 尝试解码为 CollectionTx
	var collectionTx CollectionTx
	if err := json.Unmarshal(aux.TxInner, &collectionTx); err == nil {
		tx.TxInner = collectionTx
		return nil
	}

	// 尝试解码为 MintTx
	var mintTx MintTx
	if err := json.Unmarshal(aux.TxInner, &mintTx); err == nil {
		tx.TxInner = mintTx
		return nil
	}

	// 如果没有匹配到，返回错误或解码为默认类型
	return fmt.Errorf("failed to decode TxInner")
}

// Sign signs the transaction with the private key
func (tx *Transaction) Sign(privateKey crypto.PrivateKey) error {
	// 会被计算在hash的字段，必须在hash前就赋值，不能hash完后再赋值，不然前后hash的值都不一样
	tx.From = privateKey.PublicKey()
	hash := tx.Hash(TxHasher{})
	sig, err := privateKey.Sign(hash.ToSlice())
	if err != nil {
		return err
	}

	tx.Signature = sig

	return nil
}

// Verify verifies the transaction signature
func (tx *Transaction) Verify() error {
	if tx.Signature == nil {
		return fmt.Errorf("transaction has no signature")
	}

	hash := tx.Hash(TxHasher{})
	if !tx.Signature.Verify(tx.From, hash.ToSlice()) {
		return fmt.Errorf("invalid transaction signature")
	}
	return nil
}

func (tx *Transaction) Decode(dec Decoder[*Transaction]) error {
	return dec.Decode(tx)
}

func (tx *Transaction) Encode(dec Encoder[*Transaction]) error {
	return dec.Encode(tx)
}

// Hash calculates the hash of the transaction
// todo:这里其实不是很严谨，应该明确在sign的时候，才会去赋值tx.TXHash
func (tx *Transaction) Hash(hasher Hasher[*Transaction]) types.Hash {
	if tx.TXHash.IsZero() {
		tx.TXHash = hasher.Hash(tx)
	}
	return hasher.Hash(tx)
}
