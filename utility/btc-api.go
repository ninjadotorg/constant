package utility

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

// type of timestamp in blockheader
// const API_KEY = "a2f2bad22feb460482efe5fbbefde77f"
var (
	blockTimestamp int64
	blockHash      string
)

func GetNonceByTimestamp(timestamp int64) (float64, error) {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	resp, err := http.Get("https://api.blockcypher.com/v1/btc/test3")
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		chainBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return -1, err
		}
		chain := make(map[string]interface{})
		json.Unmarshal(chainBytes, &chain)
		blockHash = chain["hash"].(string)
		// TODO: 0xmerman calculate timestamp to get the right nonce
		// get list of block with timestamp > given timestamp then get block with min timestamp value
		for true {
			_, blockTimestamp, err = getNonceOrTimeStampByBlock(blockHash, false)
			if err != nil {
				return -1, err
			}
			break
		}
		return chain["height"].(float64), err
	}
	return -1, errors.New("ERROR Getting nonce from Bitcoin")
}

//true for nonce, false for time
func getNonceOrTimeStampByBlock(blockHash string, nonceOrTime bool) (int64, int64, error) {
	resp, err := http.Get("https://api.blockcypher.com/v1/btc/test3/blocks/" + blockHash)
	if err != nil {
		return -1, -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		blockBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return -1, -1, err
		}
		block := make(map[string]interface{})
		json.Unmarshal(blockBytes, &block)
		if nonceOrTime {
			return int64(block["nonce"].(float64)), -1, nil
		} else {
			return -1, int64(block["timestamp"].(float64)), nil
		}
	}
	return -1, -1, errors.New("ERROR Getting nonce from Bitcoin")
}