package metadata

import (
	"strconv"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
)

type BuySellResponse struct {
	// for either bond or gov tokens
	MetadataBase
	RequestedTxID common.Hash

	// for buying bonds requested tx
	StartSellingAt uint64
	Maturity       uint64
	BuyBackPrice   uint64 // in Constant unit
	BondID         []byte // 24 bytes as compound value of (Maturity + BuyBackPrice + StartSellingAt) from SellingBonds param
}

func NewBuySellResponse(
	requestedTxID common.Hash,
	startSellingAt uint64,
	maturity uint64,
	buyBackPrice uint64,
	bondID []byte,
	metaType int,
) *BuySellResponse {
	metaBase := MetadataBase{
		Type: metaType,
	}
	return &BuySellResponse{
		RequestedTxID:  requestedTxID,
		StartSellingAt: startSellingAt,
		Maturity:       maturity,
		BuyBackPrice:   buyBackPrice,
		BondID:         bondID,
		MetadataBase:   metaBase,
	}
}

func (bsRes *BuySellResponse) CheckTransactionFee(tr Transaction, minFee uint64) bool {
	// no need to have fee for this tx
	return true
}

func (bsRes *BuySellResponse) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, shardID byte, db database.DatabaseInterface) (bool, error) {
	// no need to validate tx with blockchain, just need to validate with requeste tx (via RequestedTxID) in current block
	return false, nil
}

func (bsRes *BuySellResponse) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	return false, true, nil
}

func (bsRes *BuySellResponse) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	return true
}

func (bsRes *BuySellResponse) Hash() *common.Hash {
	record := bsRes.RequestedTxID.String()
	record += strconv.FormatUint(uint64(bsRes.StartSellingAt), 10)
	record += strconv.FormatUint(uint64(bsRes.Maturity), 10)
	record += strconv.FormatUint(bsRes.BuyBackPrice, 10)
	record += string(bsRes.BondID)
	record += bsRes.MetadataBase.Hash().String()

	// final hash
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (bsRes *BuySellResponse) CalculateSize() uint64 {
	return calculateSize(bsRes)
}
