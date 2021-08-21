package platypus

import (
	"context"
	"github.com/monetr/rest-api/pkg/crumbs"
	"github.com/pkg/errors"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/plaid/plaid-go/plaid"
	"github.com/sirupsen/logrus"
)

type (
	Client interface {
		GetAccounts(ctx context.Context, accountIds ...string) ([]BankAccount, error)
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

// after is a wrapper around some of the basic operations we would want to perform after each request. Mainly that we
// want to keep track of things like the request Id and some information about the request itself. It also handles error
// wrapping.
func (p *PlaidClient) after(span *sentry.Span, response *http.Response, err error, message, errorMessage string) error {
	if response != nil {
		requestId := response.Header.Get("X-Request-Id")
		span.Data["plaidRequestId"] = requestId
		span.SetTag("plaidRequestId", requestId)
		crumbs.HTTP(
			span.Context(),
			message,
			"plaid",
			response.Request.URL.String(),
			response.Request.Method,
			response.StatusCode,
			map[string]interface{}{
				"X-RequestId": requestId,
			},
		)
	}
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
	}

	span.Status = sentry.SpanStatusOK

	return errors.Wrap(err, errorMessage)
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
	if err = p.after(
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
