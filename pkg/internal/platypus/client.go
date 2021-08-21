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
	Platypus interface {
		NewClientFromItemId(ctx context.Context, itemId string)
		NewClientFromLink(ctx context.Context, accountId uint64, linkId uint64)
	}

	Client interface {
		GetAccount(ctx context.Context, accountIds ...string) ([]BankAccount, error)
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
			nil,
		)
	}
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
	}

	span.Status = sentry.SpanStatusOK

	return errors.Wrap(err, errorMessage)
}

func (p *PlaidClient) GetAccount(ctx context.Context, accountIds ...string) ([]BankAccount, error) {
	span := sentry.StartSpan(ctx, "Plaid - GetAccount")
	defer span.Finish()

	log := p.getLog(span)

	span.Data = map[string]interface{}{
		"accountIds": "ALL_BANK_ACCOUNTS",
	}

	if len(accountIds) > 0 {
		span.Data["accountIds"] = accountIds
	}

	log.Trace("retrieving bank accounts from plaid")

	request := p.client.PlaidApi.
		AccountsGet(span.Context()).
		AccountsGetRequest(plaid.AccountsGetRequest{
			AccessToken: p.accessToken,
			Options: &plaid.AccountsGetRequestOptions{
				AccountIds: &accountIds,
			},
		})

	result, response, err := request.Execute()
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
	for i, plaidAccount := range plaidAccounts {
		accounts[i], err = NewPlaidBankAccount(plaidAccount)
		if err != nil {
			log.WithError(err).WithField("bankAccountId", plaidAccount.GetAccountId()).Errorf("failed to convert bank account")
			crumbs.Error(span.Context(), "failed to convert bank account", "debug", map[string]interface{}{
				"bankAccountId": plaidAccount.GetAccountId(),
			})
			return nil, err
		}
	}

	return accounts, nil
}
