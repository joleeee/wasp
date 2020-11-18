// tx builder for VM
package statetxbuilder

import (
	"errors"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	valuetransaction "github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/wasp/packages/coretypes"
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/sctransaction"
)

type Builder struct {
	vtxb            *vtxBuilder
	chainAddress    address.Address
	stateSection    *sctransaction.StateSection
	requestSections []*sctransaction.RequestSection
}

func New(chainAddress address.Address, chainColor balance.Color, addressBalances map[valuetransaction.ID][]*balance.Balance) (*Builder, error) {
	if chainColor == balance.ColorNew || chainColor == balance.ColorIOTA {
		return nil, errors.New("statetxbuilder.New: wrong chain color")
	}
	vtxb, err := newValueTxBuilder(chainAddress, addressBalances)
	if err != nil {
		return nil, err
	}
	ret := &Builder{
		vtxb:            vtxb,
		chainAddress:    chainAddress,
		stateSection:    sctransaction.NewStateSection(sctransaction.NewStateSectionParams{Color: chainColor}),
		requestSections: make([]*sctransaction.RequestSection, 0),
	}
	err = vtxb.MoveTokens(ret.chainAddress, chainColor, 1)
	return ret, err
}

func (txb *Builder) Clone() *Builder {
	ret := &Builder{
		vtxb:            txb.vtxb.clone(),
		chainAddress:    txb.chainAddress,
		stateSection:    txb.stateSection.Clone(),
		requestSections: make([]*sctransaction.RequestSection, len(txb.requestSections)),
	}
	for i := range ret.requestSections {
		ret.requestSections[i] = txb.requestSections[i].Clone()
	}
	return ret
}

func (txb *Builder) SetStateParams(stateIndex uint32, stateHash *hashing.HashValue, timestamp int64) error {
	txb.stateSection.WithStateParams(stateIndex, stateHash, timestamp)
	return nil
}

// AddRequestSectionWithTransfer adds request block with the request
// token and adds respective outputs for the colored transfers
func (txb *Builder) AddRequestSection(req *sctransaction.RequestSection) error {
	targetAddr := address.Address(req.Target().ChainID())
	var err error
	if err = txb.vtxb.MintColor(targetAddr, balance.ColorIOTA, 1); err != nil {
		return err
	}
	if err = txb.TransferToAddress(targetAddr, req.Transfer()); err != nil {
		return err
	}
	txb.requestSections = append(txb.requestSections, req)
	return nil
}

func (txb *Builder) TransferToAddress(targetAddr address.Address, transfer coretypes.ColoredBalances) error {
	var err error
	transfer.Iterate(func(col balance.Color, bal int64) bool {
		err = txb.vtxb.MoveTokens(targetAddr, col, bal)
		if err != nil {
			return false
		}
		return true
	})
	return err
}

func (txb *Builder) Build() (*sctransaction.Transaction, error) {
	return sctransaction.NewTransaction(
		txb.vtxb.build(),
		txb.stateSection,
		txb.requestSections,
	)
}