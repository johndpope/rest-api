package platypus

import (
	"context"
	"github.com/monetr/rest-api/pkg/crumbs"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/plaid/plaid-go/plaid"
	"github.com/sirupsen/logrus"
)

type (
	Client interface {
		GetAccounts(ctx context.Context, accountIds ...string) ([]BankAccount, error)
		GetAllTransactions(ctx context.Context, start, end time.Time, accountIds []string) (interface{}, error)
		RemoveItem(ctx context.Context) error
	}
)

var (
	_ Client = &PlaidClient{}
)

type PlaidClient struct {
	accountId   uint64
	linkId      uint64
	accessToken string
	log         *logrus.Entry
	client      *plaid.APIClient
}

func (p *PlaidClient) getLog(span *sentry.Span) *logrus.Entry {
	return p.log.WithContext(span.Context()).WithField("plaid", span.Op)
}


func (p *PlaidClient) GetAccounts(ctx context.Context, accountIds ...string) ([]BankAccount, error) {
	span := sentry.StartSpan(ctx, "Plaid - GetAccount")
	defer span.Finish()

	log := p.getLog(span)

	// By default report the accountIds as "all accounts" to sentry. This way we know that if we are not requesting
	// specific accounts then we are requesting all of them.
	span.Data = map[string]interface{}{
		"accountIds": "ALL_BANK_ACCOUNTS",
	}

	// If however we are requesting specific accounts, overwrite the value.
	if len(accountIds) > 0 {
		span.Data["accountIds"] = accountIds
	}

	log.Trace("retrieving bank accounts from plaid")

	// Build the get accounts request.
	request := p.client.PlaidApi.
		AccountsGet(span.Context()).
		AccountsGetRequest(plaid.AccountsGetRequest{
			AccessToken: p.accessToken,
			Options: &plaid.AccountsGetRequestOptions{
				// This might not work, if it does not we should just add a nil check somehow here.
				AccountIds: &accountIds,
			},
		})

	// Send the request.
	result, response, err := request.Execute()
	// And handle the response.
	if err = after(
		span,
		response,
		err,
		"Retrieving bank accounts from Plaid",
		"failed to retrieve bank accounts from plaid",
	); err != nil {
		log.WithError(err).Errorf("failed to retrieve bank accounts from plaid")
		return nil, err
	}

	plaidAccounts := result.GetAccounts()
	accounts := make([]BankAccount, len(plaidAccounts))

	// Once we have our data, convert all of the results from our request to our own bank account interface.
	for i, plaidAccount := range plaidAccounts {
		accounts[i], err = NewPlaidBankAccount(plaidAccount)
		if err != nil {
			log.WithError(err).
				WithField("bankAccountId", plaidAccount.GetAccountId()).
				Errorf("failed to convert bank account")
			crumbs.Error(span.Context(), "failed to convert bank account", "debug", map[string]interface{}{
				// Maybe we don't want to report the entire account object here, but it'll sure save us a ton of time
				// if there is ever a problem with actually converting the account. This way we can actually see the
				// account object that caused the problem -> when it caused the problem.
				"bankAccount": plaidAccount,
			})
			return nil, err
		}
	}

	return accounts, nil
}

func (p *PlaidClient) GetAllTransactions(ctx context.Context, start, end time.Time, accountIds []string) (interface{}, error) {
	panic("implement me")
}

func (p *PlaidClient) RemoveItem(ctx context.Context) error {
	span := sentry.StartSpan(ctx, "Plaid - RemoveItem")
	defer span.Finish()

	log := p.getLog(span)

	log.Trace("removing item")

	// Build the get accounts request.
	request := p.client.PlaidApi.
		ItemRemove(span.Context()).
		ItemRemoveRequest(plaid.ItemRemoveRequest{
			AccessToken: p.accessToken,
		})

	// Send the request.
	_, response, err := request.Execute()
	// And handle the response.
	if err = after(
		span,
		response,
		err,
		"Removing Plaid item",
		"failed to remove Plaid item",
	); err != nil {
		log.WithError(err).Errorf("failed to remove Plaid item")
		return err
	}

	return nil
}

