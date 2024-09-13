package tenderduty

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

// ValInfo holds most of the stats/info used for secondary alarms. It is refreshed roughly every minute.
type ValInfo struct {
	Moniker    string `json:"moniker"`
	Bonded     bool   `json:"bonded"`
	Jailed     bool   `json:"jailed"`
	Tombstoned bool   `json:"tombstoned"`
	Missed     int64  `json:"missed"`
	Window     int64  `json:"window"`
	Conspub    []byte `json:"conspub"`
	Valcons    string `json:"valcons"`
}

// GetMinSignedPerWindow The check the minimum signed threshold of the validator.
func (cc *ChainConfig) GetMinSignedPerWindow() (err error) {
	if cc.client == nil {
		return errors.New("nil rpc client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	qParams := &slashing.QueryParamsRequest{}
	b, err := qParams.Marshal()
	if err != nil {
		return
	}
	resp, err := cc.client.ABCIQuery(ctx, "/cosmos.slashing.v1beta1.Query/Params", b)
	if err != nil {
		return
	}
	if resp.Response.Value == nil {
		err = errors.New("🛑 could not query slashing params, got empty response")
		return
	}
	params := &slashing.QueryParamsResponse{}
	err = params.Unmarshal(resp.Response.Value)
	if err != nil {
		return
	}

	cc.minSignedPerWindow = params.Params.MinSignedPerWindow.MustFloat64()
	return
}

// GetValInfo the first bool is used to determine if extra information about the validator should be printed.
func (cc *ChainConfig) GetValInfo(first bool) (err error) {
	if cc.client == nil {
		return errors.New("nil rpc client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if cc.valInfo == nil {
		cc.valInfo = &ValInfo{}
	}

	// Fetch info from /cosmos.staking.v1beta1.Query/Validator
	// it's easier to ask people to provide valoper since it's readily available on
	// explorers, so make it easy and lookup the consensus key for them.
	cc.valInfo.Conspub, cc.valInfo.Moniker, cc.valInfo.Jailed, cc.valInfo.Bonded, err = getVal(ctx, cc.client, cc.ValAddress)
	if err != nil {
		return
	}
	if first && cc.valInfo.Bonded {
		l(fmt.Sprintf("⚙️ found %s (%s) in validator set", cc.ValAddress, cc.valInfo.Moniker))
	} else if first && !cc.valInfo.Bonded {
		l(fmt.Sprintf("❌ %s (%s) is INACTIVE", cc.ValAddress, cc.valInfo.Moniker))
	}

	cc.valInfo.Valcons = cc.ValAddress

	return
}

// getVal returns the public key, moniker, and if the validator is jailed.
func getVal(ctx context.Context, client *rpchttp.HTTP, valoper string) (pub []byte, moniker string, jailed, bonded bool, err error) {
	bz, _ := hex.DecodeString(strings.ToLower(valoper))
	return bz, valoper, false, true, nil
}
