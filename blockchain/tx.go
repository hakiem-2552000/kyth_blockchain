package blockchain

type TxOutput struct {
	Value  int
	PubKey string
}

type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

func (in *TxInput) CanUnlock(unlockingData string) bool {
	return in.Sig == unlockingData
}

func (out *TxOutput) CanBeUnlocked(unlockingData string) bool {
	return out.PubKey == unlockingData
}
