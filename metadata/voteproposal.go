package metadata

import (
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
	"github.com/ninjadotorg/constant/privacy"
)

//abstract class
type SealedVoteProposal struct {
	SealVoteProposalData []byte
	LockerPaymentAddress []privacy.PaymentAddress
}

func NewSealedVoteProposalMetadata(sealedVoteProposal []byte, lockerPubKeys []privacy.PaymentAddress) *SealedVoteProposal {
	return &SealedVoteProposal{
		SealVoteProposalData: sealedVoteProposal,
		LockerPaymentAddress: lockerPubKeys,
	}
}

func (sealedVoteProposal *SealedVoteProposal) Hash2() *common.Hash {
	record := string(sealedVoteProposal.SealVoteProposalData)
	for _, i := range sealedVoteProposal.LockerPaymentAddress {
		record += string(i.Bytes())
	}
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealedVoteProposal *SealedVoteProposal) ValidateLockerPubKeys(bcr BlockchainRetriever, boardType string) (bool, error) {
	//Validate these pubKeys are in board
	boardPaymentAddress := bcr.GetBoardPaymentAddress(boardType)
	for _, j := range sealedVoteProposal.LockerPaymentAddress {
		exist := common.FalseValue
		for _, i := range boardPaymentAddress {
			if common.ByteEqual(i.Bytes(), j.Bytes()) {
				exist = common.TrueValue
				break
			}
		}
		if !exist {
			return common.FalseValue, nil
		}
	}
	return common.TrueValue, nil
}

func (sealedVoteProposal *SealedVoteProposal) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	return common.TrueValue, common.TrueValue, nil
}

func (sealedVoteProposal *SealedVoteProposal) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(sealedVoteProposal.LockerPaymentAddress); index1++ {
		pub1 := sealedVoteProposal.LockerPaymentAddress[index1]
		for index2 := index1 + 1; index2 < len(sealedVoteProposal.LockerPaymentAddress); index2++ {
			pub2 := sealedVoteProposal.LockerPaymentAddress[index2]
			if !common.ByteEqual(pub1.Bytes(), pub2.Bytes()) {
				return common.FalseValue
			}
		}
	}
	return common.TrueValue
}

type SealedLv1VoteProposalMetadata struct {
	SealedVoteProposal       SealedVoteProposal
	PointerToLv2VoteProposal common.Hash
	PointerToLv3VoteProposal common.Hash
}

func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) GetBoardType() string {
	// TODO: @0xjackalope
	panic("override me")
}
func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) ValidataBeforeNewBlock(tx Transaction, bcr BlockchainRetriever, chainID byte) bool {
	boardType := sealedLv1VoteProposalMetadata.GetBoardType()
	endedPivot := bcr.GetConstitutionEndHeight(boardType, chainID)
	currentBlockHeight := bcr.GetCurrentBlockHeight(chainID) + 1
	lv3Pivot := endedPivot - uint64(common.EncryptionOnePhraseDuration)
	lv2Pivot := lv3Pivot - uint64(common.EncryptionOnePhraseDuration)
	lv1Pivot := lv2Pivot - uint64(common.EncryptionOnePhraseDuration)
	if !(currentBlockHeight < lv1Pivot && currentBlockHeight >= lv2Pivot) {
		return common.FalseValue
	}
	return common.TrueValue
}

func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) ValidateSanityData(bcr BlockchainRetriever, tx Transaction) (bool, bool, error) {
	_, ok, _ := sealedLv1VoteProposalMetadata.SealedVoteProposal.ValidateSanityData(bcr, tx)
	if !ok {
		return common.TrueValue, common.FalseValue, nil
	}
	return common.TrueValue, common.TrueValue, nil
}

func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) ValidateMetadataByItself() bool {
	return common.TrueValue
}

func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) ValidateTxWithBlockChain(boardType string, transaction Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv1VoteProposalMetadata.SealedVoteProposal.ValidateLockerPubKeys(bcr, boardType)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv2Tx, _ := bcr.GetTransactionByHash(&sealedLv1VoteProposalMetadata.PointerToLv2VoteProposal)
	if lv2Tx.GetMetadataType() != GetSealedLv2VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&sealedLv1VoteProposalMetadata.PointerToLv3VoteProposal)
	if lv3Tx.GetMetadataType() != GetSealedLv3VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}

	// check 2 array equal
	sealLv2VoteProposalMetadata := GetSealedLv2VoteProposalMetadata(lv2Tx, boardType)
	for i := 0; i < len(sealedLv1VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress); i++ {
		if !common.ByteEqual(sealedLv1VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes(), sealLv2VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes()) {
			return common.FalseValue, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(sealedLv1VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData,
		common.Encrypt(sealLv2VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData, sealLv2VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[1].Pk)) {
		return common.FalseValue, nil
	}
	return common.TrueValue, nil
}

func GetSealedLv2VoteProposalMetadata(transaction Transaction, boardType string) SealedLv2VoteProposalMetadata {
	meta := transaction.GetMetadata()
	if boardType == "dcb" {
		return meta.(*SealedLv2DCBVoteProposalMetadata).SealedLv2VoteProposalMetadata
	} else {
		return meta.(*SealedLv2GOVVoteProposalMetadata).SealedLv2VoteProposalMetadata
	}
}

func GetSealedLv3VoteProposalMeta(boardType string) int {
	if boardType == "dcb" {
		return SealedLv3DCBVoteProposalMeta
	} else {
		return SealedLv3GOVVoteProposalMeta
	}
}

func GetSealedLv2VoteProposalMeta(boardType string) int {
	if boardType == "dcb" {
		return SealedLv2DCBVoteProposalMeta
	} else {
		return SealedLv2GOVVoteProposalMeta
	}

}

func NewSealedLv1VoteProposalMetadata(
	sealedVoteProposal []byte,
	lockersPaymentAddress []privacy.PaymentAddress,
	pointerToLv2VoteProposal common.Hash,
	pointerToLv3VoteProposal common.Hash,
) *SealedLv1VoteProposalMetadata {
	return &SealedLv1VoteProposalMetadata{
		SealedVoteProposal:       *NewSealedVoteProposalMetadata(sealedVoteProposal, lockersPaymentAddress),
		PointerToLv2VoteProposal: pointerToLv2VoteProposal,
		PointerToLv3VoteProposal: pointerToLv3VoteProposal,
	}
}

func (sealedLv1VoteProposalMetadata *SealedLv1VoteProposalMetadata) ToBytes() []byte {
	record := string(sealedLv1VoteProposalMetadata.SealedVoteProposal.Hash2().GetBytes())
	record += string(sealedLv1VoteProposalMetadata.PointerToLv2VoteProposal.GetBytes())
	record += string(sealedLv1VoteProposalMetadata.PointerToLv3VoteProposal.GetBytes())
	return []byte(record)
}

type SealedLv2VoteProposalMetadata struct {
	SealedVoteProposal
	PointerToLv3VoteProposal common.Hash
}

func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) ToBytes() []byte {
	record := string(sealedLv2VoteProposalMetadata.SealedVoteProposal.Hash2().GetBytes())
	record += string(sealedLv2VoteProposalMetadata.PointerToLv3VoteProposal.GetBytes())
	return []byte(record)
}

func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) GetBoardType() string {
	panic("overwrite me")
}
func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) ValidataBeforeNewBlock(tx Transaction, bcr BlockchainRetriever, chainID byte) bool {
	boardType := sealedLv2VoteProposalMetadata.GetBoardType()
	endedPivot := bcr.GetConstitutionEndHeight(boardType, chainID)
	currentBlockHeight := bcr.GetCurrentBlockHeight(chainID) + 1
	lv3Pivot := endedPivot - uint64(common.EncryptionOnePhraseDuration)
	lv2Pivot := lv3Pivot - uint64(common.EncryptionOnePhraseDuration)
	if !(currentBlockHeight < lv2Pivot && currentBlockHeight >= lv3Pivot) {
		return common.FalseValue
	}
	return common.TrueValue
}

func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) ValidateSanityData(bcr BlockchainRetriever, tx Transaction) (bool, bool, error) {
	_, ok, _ := sealedLv2VoteProposalMetadata.SealedVoteProposal.ValidateSanityData(bcr, tx)
	if !ok {
		return common.TrueValue, common.FalseValue, nil
	}
	return common.TrueValue, common.TrueValue, nil
}

func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) ValidateMetadataByItself() bool {
	return common.TrueValue
}

func (sealedLv2VoteProposalMetadata *SealedLv2VoteProposalMetadata) ValidateTxWithBlockChain(
	boardType string,
	transaction Transaction,
	bcr BlockchainRetriever,
	chainID byte,
	db database.DatabaseInterface,
) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv2VoteProposalMetadata.SealedVoteProposal.ValidateLockerPubKeys(bcr, boardType)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&sealedLv2VoteProposalMetadata.PointerToLv3VoteProposal)
	if lv3Tx.GetMetadataType() != GetSealedLv3VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}

	// check 2 array equal
	sealedLv3VoteProposalMetadata := GetSealedLv3VoteProposalMetadata(boardType, lv3Tx)
	for i := 0; i < len(sealedLv2VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress); i++ {
		if !common.ByteEqual(
			sealedLv2VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes(),
			sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes(),
		) {
			return common.FalseValue, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(
		sealedLv2VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData,
		common.Encrypt(sealedLv3VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData,
			sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[2].Pk,
		),
	) {
		return common.FalseValue, nil
	}
	return common.TrueValue, nil
}

func GetSealedLv3VoteProposalMetadata(boardType string, transaction Transaction) SealedLv3VoteProposalMetadata {
	meta := transaction.GetMetadata()
	if boardType == "dcb" {
		return meta.(*SealedLv3DCBVoteProposalMetadata).SealedLv3VoteProposalMetadata
	} else {
		return meta.(*SealedLv3GOVVoteProposalMetadata).SealedLv3VoteProposalMetadata
	}

}

func NewSealedLv2VoteProposalMetadata(
	sealedVoteProposal []byte,
	lockerPaymentAddress []privacy.PaymentAddress,
	pointerToLv3VoteProposal common.Hash,
) *SealedLv2VoteProposalMetadata {
	return &SealedLv2VoteProposalMetadata{
		SealedVoteProposal: *NewSealedVoteProposalMetadata(
			sealedVoteProposal,
			lockerPaymentAddress,
		),
		PointerToLv3VoteProposal: pointerToLv3VoteProposal,
	}

}

type SealedLv3VoteProposalMetadata struct {
	SealedVoteProposal SealedVoteProposal
}

func (sealedLv3VoteProposalMetadata *SealedLv3VoteProposalMetadata) ValidataBeforeNewBlock(boardType string, tx Transaction, bcr BlockchainRetriever, chainID byte) bool {
	startedPivot := bcr.GetConstitutionStartHeight(boardType, chainID)
	endedPivot := bcr.GetConstitutionEndHeight(boardType, chainID)
	currentBlockHeight := bcr.GetCurrentBlockHeight(chainID) + 1
	lv3Pivot := endedPivot - uint64(common.EncryptionOnePhraseDuration)
	if !(currentBlockHeight < lv3Pivot && currentBlockHeight >= startedPivot) {
		return common.FalseValue
	}
	return common.TrueValue
}

func (sealLv3VoteProposalMetadata *SealedLv3VoteProposalMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, b byte, db database.DatabaseInterface) (bool, error) {
	return sealLv3VoteProposalMetadata.ValidateTxWithBlockChain(tx, bcr, b, db)
}

func (sealLv3VoteProposalMetadata *SealedLv3VoteProposalMetadata) ValidateSanityData(bcr BlockchainRetriever, tx Transaction) (bool, bool, error) {
	return sealLv3VoteProposalMetadata.ValidateSanityData(bcr, tx)
}

func (sealLv3VoteProposalMetadata *SealedLv3VoteProposalMetadata) ValidateMetadataByItself() bool {
	return sealLv3VoteProposalMetadata.ValidateMetadataByItself()
}

func NewSealedLv3VoteProposalMetadata(
	sealedVoteProposal []byte,
	lockerPaymentAddress []privacy.PaymentAddress,
) *SealedLv3VoteProposalMetadata {
	return &SealedLv3VoteProposalMetadata{
		SealedVoteProposal: *NewSealedVoteProposalMetadata(sealedVoteProposal, lockerPaymentAddress),
	}

}

type VoteProposalData struct {
	ProposalTxID common.Hash
	AmountOfVote int32
}

func NewVoteProposalData(proposalTxID common.Hash, amountOfVote int32) *VoteProposalData {
	return &VoteProposalData{ProposalTxID: proposalTxID, AmountOfVote: amountOfVote}
}

func NewVoteProposalDataFromJson(data interface{}) *VoteProposalData {
	voteProposalDataData := data.(map[string]interface{})
	return NewVoteProposalData(
		common.NewHash([]byte(voteProposalDataData["ProposalTxID"].(string))),
		int32(voteProposalDataData["AmountOfVote"].(float64)),
	)
}

func (voteProposalData VoteProposalData) ToBytes() []byte {
	b := voteProposalData.ProposalTxID.GetBytes()
	b = append(b, common.Int32ToBytes(voteProposalData.AmountOfVote)...)
	return b
}

func NewVoteProposalDataFromBytes(b []byte) *VoteProposalData {
	lenB := len(b)
	return NewVoteProposalData(
		common.NewHash(b[:lenB-4]),
		common.BytesToInt32(b[lenB-4:]),
	)
}

func (voteProposalData VoteProposalData) Hash2() *common.Hash {
	record := string(voteProposalData.ProposalTxID.GetBytes())
	record += string(voteProposalData.AmountOfVote)

	hash := common.DoubleHashH([]byte(record))
	return &hash
}

type NormalVoteProposalFromSealerMetadata struct {
	VoteProposal             VoteProposalData
	LockerPaymentAddress     []privacy.PaymentAddress
	PointerToLv1VoteProposal common.Hash
	PointerToLv3VoteProposal common.Hash
}

func NewNormalVoteProposalFromSealerMetadata(
	voteProposal VoteProposalData,
	lockerPaymentAddress []privacy.PaymentAddress,
	pointerToLv1VoteProposal common.Hash,
	pointerToLv3VoteProposal common.Hash,
) *NormalVoteProposalFromSealerMetadata {
	return &NormalVoteProposalFromSealerMetadata{
		VoteProposal:             voteProposal,
		LockerPaymentAddress:     lockerPaymentAddress,
		PointerToLv1VoteProposal: pointerToLv1VoteProposal,
		PointerToLv3VoteProposal: pointerToLv3VoteProposal,
	}
}
func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) GetBoardType() string {
	panic("overwrite me")
}
func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	return common.TrueValue, common.TrueValue, nil
}

func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(normalVoteProposalFromSealerMetadata.LockerPaymentAddress); index1++ {
		pub1 := normalVoteProposalFromSealerMetadata.LockerPaymentAddress[index1]
		for index2 := index1 + 1; index2 < len(normalVoteProposalFromSealerMetadata.LockerPaymentAddress); index2++ {
			pub2 := normalVoteProposalFromSealerMetadata.LockerPaymentAddress[index2]
			if !common.ByteEqual(pub1.Bytes(), pub2.Bytes()) {
				return common.FalseValue
			}
		}
	}
	return common.TrueValue
}

func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) ToBytes() []byte {
	record := string(normalVoteProposalFromSealerMetadata.VoteProposal.Hash2().GetBytes())
	for _, i := range normalVoteProposalFromSealerMetadata.LockerPaymentAddress {
		record += string(i.Bytes())
	}
	record += string(normalVoteProposalFromSealerMetadata.PointerToLv1VoteProposal.GetBytes())
	record += string(normalVoteProposalFromSealerMetadata.PointerToLv3VoteProposal.GetBytes())
	return []byte(record)
}

func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) ValidataBeforeNewBlock(tx Transaction, bcr BlockchainRetriever, chainID byte) bool {
	boardType := normalVoteProposalFromSealerMetadata.GetBoardType()
	endedPivot := bcr.GetConstitutionEndHeight(boardType, chainID)
	currentBlockHeight := bcr.GetCurrentBlockHeight(chainID) + 1
	lv3Pivot := endedPivot - uint64(common.EncryptionOnePhraseDuration)
	lv2Pivot := lv3Pivot - uint64(common.EncryptionOnePhraseDuration)
	lv1Pivot := lv2Pivot - uint64(common.EncryptionOnePhraseDuration)
	if !(currentBlockHeight < endedPivot && currentBlockHeight >= lv1Pivot) {
		return common.FalseValue
	}
	return common.TrueValue
}

func (normalVoteProposalFromSealerMetadata *NormalVoteProposalFromSealerMetadata) ValidateTxWithBlockChain(boardType string,
	transaction Transaction,
	bcr BlockchainRetriever,
	chainID byte,
	db database.DatabaseInterface) (bool, error) {
	boardPubKeys := bcr.GetBoardPaymentAddress(boardType)
	for _, j := range normalVoteProposalFromSealerMetadata.LockerPaymentAddress {
		exist := common.FalseValue
		for _, i := range boardPubKeys {
			if common.ByteEqual(i.Bytes(), j.Bytes()) {
				exist = common.TrueValue
				break
			}
		}
		if !exist {
			return common.FalseValue, nil
		}
	}

	//Check precede transaction type
	_, _, _, lv1Tx, _ := bcr.GetTransactionByHash(&normalVoteProposalFromSealerMetadata.PointerToLv1VoteProposal)
	if lv1Tx.GetMetadataType() != GetSealedLv1VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&normalVoteProposalFromSealerMetadata.PointerToLv3VoteProposal)
	if lv3Tx.GetMetadataType() != GetSealedLv3VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}

	// check 2 array equal
	sealedLv1VoteProposalMetadata := GetSealedLv1VoteProposalMetadata(boardType, lv1Tx)
	for i := 0; i < len(normalVoteProposalFromSealerMetadata.LockerPaymentAddress); i++ {
		if !common.ByteEqual(normalVoteProposalFromSealerMetadata.LockerPaymentAddress[i].Bytes(), sealedLv1VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes()) {
			return common.FalseValue, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(normalVoteProposalFromSealerMetadata.VoteProposal.ToBytes(),
		common.Encrypt(
			sealedLv1VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData,
			sealedLv1VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[0].Pk,
		)) {
		return common.FalseValue, nil
	}
	return common.TrueValue, nil
}

func GetSealedLv1VoteProposalMetadata(boardType string, transaction Transaction) SealedLv1VoteProposalMetadata {
	meta := transaction.GetMetadata()
	if boardType == "dcb" {
		return meta.(*SealedLv1DCBVoteProposalMetadata).SealedLv1VoteProposalMetadata
	} else {
		return meta.(*SealedLv1GOVVoteProposalMetadata).SealedLv1VoteProposalMetadata
	}
}

func GetSealedLv1VoteProposalMeta(boardType string) int {
	if boardType == "dcb" {
		return SealedLv1DCBVoteProposalMeta
	} else {
		return SealedLv1GOVVoteProposalMeta
	}
}

type NormalVoteProposalFromOwnerMetadata struct {
	VoteProposal             VoteProposalData
	LockerPaymentAddress     []privacy.PaymentAddress
	PointerToLv3VoteProposal common.Hash
}

func NewNormalVoteProposalFromOwnerMetadata(
	voteProposal VoteProposalData,
	lockerPaymentAddress []privacy.PaymentAddress,
	pointerToLv3VoteProposal common.Hash,
) *NormalVoteProposalFromOwnerMetadata {
	return &NormalVoteProposalFromOwnerMetadata{
		VoteProposal:             voteProposal,
		LockerPaymentAddress:     lockerPaymentAddress,
		PointerToLv3VoteProposal: pointerToLv3VoteProposal,
	}
}

func (normalVoteProposalFromOwnerMetadata *NormalVoteProposalFromOwnerMetadata) ValidataBeforeNewBlock(boardType string, tx Transaction, bcr BlockchainRetriever, chainID byte) bool {
	endedPivot := bcr.GetConstitutionEndHeight(boardType, chainID)
	currentBlockHeight := bcr.GetCurrentBlockHeight(chainID) + 1
	lv3Pivot := endedPivot - common.EncryptionOnePhraseDuration
	lv2Pivot := lv3Pivot - common.EncryptionOnePhraseDuration
	lv1Pivot := lv2Pivot - common.EncryptionOnePhraseDuration
	if !(currentBlockHeight < endedPivot && currentBlockHeight >= lv1Pivot) {
		return common.FalseValue
	}
	return common.TrueValue
}

func (normalVoteProposalFromOwnerMetadata *NormalVoteProposalFromOwnerMetadata) ToBytes() []byte {
	record := string(normalVoteProposalFromOwnerMetadata.VoteProposal.Hash2().GetBytes())
	for _, i := range normalVoteProposalFromOwnerMetadata.LockerPaymentAddress {
		record += string(i.Bytes())
	}
	record += string(normalVoteProposalFromOwnerMetadata.PointerToLv3VoteProposal.GetBytes())
	return []byte(record)
}

func (normalVoteProposalFromOwnerMetadata *NormalVoteProposalFromOwnerMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	return common.TrueValue, common.TrueValue, nil
}

func (normalVoteProposalFromOwnerMetadata *NormalVoteProposalFromOwnerMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(normalVoteProposalFromOwnerMetadata.LockerPaymentAddress); index1++ {
		pub1 := normalVoteProposalFromOwnerMetadata.LockerPaymentAddress[index1]
		for index2 := index1 + 1; index2 < len(normalVoteProposalFromOwnerMetadata.LockerPaymentAddress); index2++ {
			pub2 := normalVoteProposalFromOwnerMetadata.LockerPaymentAddress[index2]
			if !common.ByteEqual(pub1.Bytes(), pub2.Bytes()) {
				return common.FalseValue
			}
		}
	}
	return common.TrueValue
}

func (normalVoteProposalFromOwnerMetadata *NormalVoteProposalFromOwnerMetadata) ValidateTxWithBlockChain(
	boardType string,
	transaction Transaction,
	bcr BlockchainRetriever,
	chainID byte,
	db database.DatabaseInterface) (bool,
	error) {
	boardPaymentAddress := bcr.GetBoardPaymentAddress(boardType)
	for _, j := range normalVoteProposalFromOwnerMetadata.LockerPaymentAddress {
		exist := common.FalseValue
		for _, i := range boardPaymentAddress {
			if common.ByteEqual(i.Bytes(), j.Bytes()) {
				exist = common.TrueValue
				break
			}
		}
		if !exist {
			return common.FalseValue, nil
		}
	}

	//Check precede transaction type
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&normalVoteProposalFromOwnerMetadata.PointerToLv3VoteProposal)
	if lv3Tx.GetMetadataType() != GetSealedLv3VoteProposalMeta(boardType) {
		return common.FalseValue, nil
	}

	// check 2 array equal
	sealedLv3VoteProposalMetadata := GetSealedLv3VoteProposalMetadata(boardType, lv3Tx)
	for i := 0; i < len(normalVoteProposalFromOwnerMetadata.LockerPaymentAddress); i++ {
		if !common.ByteEqual(normalVoteProposalFromOwnerMetadata.LockerPaymentAddress[i].Bytes(),
			sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[i].Bytes(),
		) {
			return common.FalseValue, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(
		sealedLv3VoteProposalMetadata.SealedVoteProposal.SealVoteProposalData,
		common.Encrypt(
			common.Encrypt(
				common.Encrypt(
					normalVoteProposalFromOwnerMetadata.VoteProposal.ToBytes(),
					sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[2].Pk,
				),
				sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[1].Pk,
			),
			sealedLv3VoteProposalMetadata.SealedVoteProposal.LockerPaymentAddress[0].Pk,
		)) {
		return common.FalseValue, nil
	}
	return common.TrueValue, nil
}

type PunishDecryptMetadata struct {
	PaymentAddress privacy.PaymentAddress
}

func (punishDecryptMetadata PunishDecryptMetadata) ToBytes() []byte {
	return punishDecryptMetadata.PaymentAddress.Bytes()
}