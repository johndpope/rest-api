package platypus

import (
	"github.com/plaid/plaid-go/plaid"
	"time"
)

type ItemToken struct {
	AccessToken string
	ItemId      string
}

func NewItemTokenFromPlaid(input plaid.ItemPublicTokenExchangeResponse) (ItemToken, error) {
	return ItemToken{
		AccessToken: input.GetAccessToken(),
		ItemId:      input.GetItemId(),
	}, nil
}

type LinkToken struct {
	LinkToken  string
	Expiration time.Time
}
