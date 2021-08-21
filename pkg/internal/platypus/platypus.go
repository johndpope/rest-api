package platypus

import (
	"context"
	"github.com/monetr/rest-api/pkg/secrets"
	"github.com/plaid/plaid-go/plaid"
	"github.com/sirupsen/logrus"
)

type (
	Platypus interface {
		NewClientFromItemId(ctx context.Context, itemId string) (Client, error)
		NewClientFromLink(ctx context.Context, accountId uint64, linkId uint64) (Client, error)
	}
)

var (
	_ Platypus = &Plaid{}
)

type Plaid struct {
	client *plaid.APIClient
	log    *logrus.Entry
	secret secrets.PlaidSecretsProvider
}

func (p *Plaid) NewClientFromItemId(ctx context.Context, itemId string) (Client, error) {
	panic("implement me")
}

func (p *Plaid) NewClientFromLink(ctx context.Context, accountId uint64, linkId uint64) (Client, error) {
	panic("implement me")
}

