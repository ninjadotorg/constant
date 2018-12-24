package zkp

import (
	"crypto/rand"
	"errors"
	"math/big"

	privacy "github.com/ninjadotorg/constant/privacy"
)

//Openings protocol: https://courses.cs.ut.ee/MTAT.07.003/2017_fall/uploads/Main/0907-sigma-protocol-for-pedersen-commitment.pdf

// PKComOpeningsProof contains PoK
type PKComOpeningsProof struct {
	commitmentValue *privacy.EllipticPoint //statement
	indexs          []byte                 //statement
	alpha           *privacy.EllipticPoint
	gamma           []*big.Int
}

// PKComOpeningsWitness contains witnesses which are used for generate proof
type PKComOpeningsWitness struct {
	commitmentValue *privacy.EllipticPoint //statement
	indexs          []byte                 //statement
	Openings        []*big.Int
}

// Init create PKComOpeningsProof element with default value
func (pro *PKComOpeningsProof) Init() *PKComOpeningsProof {
	pro.commitmentValue = new(privacy.EllipticPoint).Zero()
	return pro
}

// IsNil return true if one of each field is null
func (pro *PKComOpeningsProof) IsNil() bool {
	if (len(pro.gamma) == 0) || (pro.commitmentValue == nil) || (pro.alpha == nil) || (pro.indexs == nil) || (pro.gamma == nil) {
		return true
	}
	return false
}

// randValue return random witness value for testing
func (wit *PKComOpeningsWitness) randValue(testcase bool) {
	wit.Openings = make([]*big.Int, privacy.PedCom.Capacity-1)
	for i := 0; i < privacy.PedCom.Capacity-1; i++ {
		wit.Openings[i], _ = rand.Int(rand.Reader, privacy.Curve.Params().N)
	}
	commit1 := privacy.PedCom.CommitAtIndex(wit.Openings[0], wit.Openings[3], 0)
	commit2 := privacy.PedCom.CommitAtIndex(wit.Openings[1], wit.Openings[3], 1)
	commit3 := privacy.PedCom.CommitAtIndex(wit.Openings[2], wit.Openings[3], 2)
	wit.commitmentValue = commit1.Add(commit2)
	wit.Openings[3].Mul(wit.Openings[3], big.NewInt(3))
	wit.Openings[3].Mod(wit.Openings[3], privacy.Curve.Params().N)
	wit.commitmentValue = wit.commitmentValue.Add(commit3)
	// wit.commitmentValue = privacy.PedCom.CommitAll(wit.Openings)
	wit.indexs = []byte{privacy.SK, privacy.VALUE, privacy.SND, privacy.RAND}
}

// Set dosomethings
func (wit *PKComOpeningsWitness) Set(
	commitmentValue *privacy.EllipticPoint, //statement
	openings []*big.Int,
	indexs []byte) {
	wit.commitmentValue = commitmentValue
	wit.Openings = openings
	wit.indexs = indexs
}

// Set dosomethings
func (pro *PKComOpeningsProof) Set(
	commitmentValue *privacy.EllipticPoint, //statement
	alpha *privacy.EllipticPoint,
	gamma []*big.Int,
	indexs []byte) {
	if pro == nil {
		pro = new(PKComOpeningsProof)
	}
	pro.commitmentValue = commitmentValue
	pro.alpha = alpha
	pro.gamma = gamma
	pro.indexs = indexs
}

// Bytes convert PKComOpeningsProof's value to byte array and return
func (pro PKComOpeningsProof) Bytes() []byte {
	if pro.IsNil() {
		return []byte{}
	}

	res := append(pro.commitmentValue.Compress(), pro.alpha.Compress()...)
	for i := 0; i < len(pro.gamma); i++ {
		temp := pro.gamma[i].Bytes()
		for j := 0; j < privacy.BigIntSize-len(temp); j++ {
			temp = append([]byte{0}, temp...)
		}
		res = append(res, temp...)
	}
	for i := 0; i < len(pro.indexs); i++ {
		res = append(res, []byte{pro.indexs[i]}...)
	}
	return res
}

// SetBytes convert byte array to PKComOpeningsProof
func (pro *PKComOpeningsProof) SetBytes(bytestr []byte) error {
	if len(bytestr) == 0 {
		return nil
	}
	pro.commitmentValue = new(privacy.EllipticPoint)
	pro.commitmentValue.Decompress(bytestr[0:privacy.CompressedPointSize])
	if !pro.commitmentValue.IsSafe() {
		return  errors.New("Decompressed failed!")
	}

	pro.alpha = new(privacy.EllipticPoint)
	pro.alpha.Decompress(bytestr[privacy.CompressedPointSize : privacy.CompressedPointSize*2])
	if !pro.alpha.IsSafe() {
		return errors.New("Decompressed failed!")
	}

	pro.gamma = make([]*big.Int, (len(bytestr)-privacy.CompressedPointSize*2)/privacy.BigIntSize)
	for i := 0; i < len(pro.gamma); i++ {
		pro.gamma[i] = big.NewInt(0)
		pro.gamma[i].SetBytes(bytestr[privacy.CompressedPointSize*2+i*privacy.BigIntSize : privacy.CompressedPointSize*2+(i+1)*privacy.BigIntSize])
	}
	pro.indexs = make([]byte, len(pro.gamma))
	for i := 0; i < len(pro.indexs); i++ {
		pro.indexs[i] = bytestr[privacy.CompressedPointSize*2+len(pro.gamma)*privacy.BigIntSize+i]
	}
	return nil
}

// Prove ... (for sender)
func (wit *PKComOpeningsWitness) Prove() (*PKComOpeningsProof, error) {
	// r1Rand, _ := rand.Int(rand.Reader, privacy.Curve.Params().N)
	// r2Rand, _ := rand.Int(rand.Reader, privacy.Curve.Params().N)
	alpha := new(privacy.EllipticPoint)
	alpha.X = big.NewInt(0)
	alpha.Y = big.NewInt(0)
	beta := GenerateChallengeFromPoint([]*privacy.EllipticPoint{wit.commitmentValue})
	gamma := make([]*big.Int, len(wit.Openings))
	//var gPowR privacy.EllipticPoint
	gPowR := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	for i := 0; i < len(wit.Openings); i++ {
		rRand, _ := rand.Int(rand.Reader, privacy.Curve.Params().N)
		gPowR.X, gPowR.Y = privacy.Curve.ScalarMult(privacy.PedCom.G[wit.indexs[i]].X, privacy.PedCom.G[wit.indexs[i]].Y, rRand.Bytes())
		alpha.X, alpha.Y = privacy.Curve.Add(alpha.X, alpha.Y, gPowR.X, gPowR.Y)
		gamma[i] = big.NewInt(0).Mul(wit.Openings[i], beta)
		gamma[i] = gamma[i].Add(gamma[i], rRand)
		gamma[i] = gamma[i].Mod(gamma[i], privacy.Curve.Params().N)
	}
	proof := new(PKComOpeningsProof)
	proof.Set(wit.commitmentValue, alpha, gamma, wit.indexs)
	return proof, nil
}

// Verify ... (for verifier)
func (pro *PKComOpeningsProof) Verify() bool {
	beta := GenerateChallengeFromPoint([]*privacy.EllipticPoint{pro.commitmentValue})
	rightPoint := new(privacy.EllipticPoint)
	rightPoint.X, rightPoint.Y = privacy.Curve.ScalarMult(pro.commitmentValue.X, pro.commitmentValue.Y, beta.Bytes())
	rightPoint.X, rightPoint.Y = privacy.Curve.Add(rightPoint.X, rightPoint.Y, pro.alpha.X, pro.alpha.Y)
	leftPoint := new(privacy.EllipticPoint)
	leftPoint.X = big.NewInt(0)
	leftPoint.Y = big.NewInt(0)
	var gPowR privacy.EllipticPoint
	for i := 0; i < len(pro.gamma); i++ {
		gPowR.X, gPowR.Y = privacy.Curve.ScalarMult(privacy.PedCom.G[pro.indexs[i]].X, privacy.PedCom.G[pro.indexs[i]].Y, pro.gamma[i].Bytes())
		leftPoint.X, leftPoint.Y = privacy.Curve.Add(leftPoint.X, leftPoint.Y, gPowR.X, gPowR.Y)
	}
	return leftPoint.IsEqual(rightPoint)
}
