package platypus

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/monetr/rest-api/pkg/config"
	"github.com/monetr/rest-api/pkg/crumbs"
	"github.com/monetr/rest-api/pkg/models"
	"github.com/monetr/rest-api/pkg/repository"
	"github.com/monetr/rest-api/pkg/secrets"
	"github.com/pkg/errors"
	"github.com/plaid/plaid-go/plaid"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type (
	Platypus interface {
		CreateLinkToken(ctx context.Context) (*LinkToken, error)
		ExchangePublicToken(ctx context.Context, publicToken string) (*ItemToken, error)
		GetWebhookVerificationKey(ctx context.Context, keyId string) (*WebhookVerificationKey, error)
		NewClientFromItemId(ctx context.Context, itemId string) (Client, error)
		NewClientFromLink(ctx context.Context, accountId uint64, linkId uint64) (Client, error)
		NewClient(ctx context.Context, link *models.Link, accessToken string) (Client, error)
		Close() error
	}
)

// after is a wrapper around some of the basic operations we would want to perform after each request. Mainly that we
// want to keep track of things like the request Id and some information about the request itself. It also handles error
// wrapping.
func after(span *sentry.Span, response *http.Response, err error, message, errorMessage string) error {
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

var (
	_ Platypus = &Plaid{}
)

func NewPlaid(log *logrus.Entry, secret secrets.PlaidSecretsProvider, repo repository.PlaidRepository, options config.Plaid) *Plaid {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	conf := plaid.NewConfiguration()
	conf.HTTPClient = httpClient
	conf.UseEnvironment(options.Environment)
	conf.AddDefaultHeader("PLAID-CLIENT-ID", options.ClientID)
	conf.AddDefaultHeader("PLAID-SECRET", options.ClientSecret)

	client := plaid.NewAPIClient(conf)

	return &Plaid{
		client: client,
		log:    log,
		secret: secret,
		repo:   repo,
	}
}

type Plaid struct {
	client *plaid.APIClient
	log    *logrus.Entry
	secret secrets.PlaidSecretsProvider
	repo   repository.PlaidRepository
}

func (p *Plaid) CreateLinkToken(ctx context.Context) (*LinkToken, error) {
	span := sentry.StartSpan(ctx, "Plaid - CreateLinkToken")
	defer span.Finish()

	log := p.log

	request := p.client.PlaidApi.
		LinkTokenCreate(span.Context()).
		LinkTokenCreateRequest(plaid.LinkTokenCreateRequest{
			ClientName:            "",
			Language:              "",
			CountryCodes:          nil,
			User:                  plaid.LinkTokenCreateRequestUser{},
			Products:              nil,
			Webhook:               nil,
			AccessToken:           nil,
			LinkCustomizationName: nil,
			RedirectUri:           nil,
			AndroidPackageName:    nil,
			AccountFilters:        nil,
			EuConfig:              nil,
			InstitutionId:         nil,
			PaymentInitiation:     nil,
			DepositSwitch:         nil,
			IncomeVerification:    nil,
			Auth:                  nil,
		})

	result, response, err := request.Execute()
	if err = after(
		span,
		response,
		err,
		"Creating link token with Plaid",
		"failed to create link token",
	); err != nil {
		log.WithError(err).Errorf("failed to create link token")
		return nil, err
	}

	fmt.Sprint(result)

	panic("not implemented")
	return nil, nil
}

func (p *Plaid) ExchangePublicToken(ctx context.Context, publicToken string) (*ItemToken, error) {
	span := sentry.StartSpan(ctx, "Plaid - ExchangePublicToken")
	defer span.Finish()

	log := p.log

	request := p.client.PlaidApi.
		ItemPublicTokenExchange(span.Context()).
		ItemPublicTokenExchangeRequest(plaid.ItemPublicTokenExchangeRequest{
			PublicToken: publicToken,
		})

	result, response, err := request.Execute()
	if err = after(
		span,
		response,
		err,
		"Exchanging public token with Plaid",
		"failed to exchange public token with Plaid",
	); err != nil {
		log.WithError(err).Errorf("failed to exchange public token with Plaid")
		return nil, err
	}

	token, err := NewItemTokenFromPlaid(result)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (p *Plaid) GetWebhookVerificationKey(ctx context.Context, keyId string) (*WebhookVerificationKey, error) {
	span := sentry.StartSpan(ctx, "Plaid - GetWebhookVerificationKey")
	defer span.Finish()

	log := p.log

	request := p.client.PlaidApi.
		WebhookVerificationKeyGet(span.Context()).
		WebhookVerificationKeyGetRequest(plaid.WebhookVerificationKeyGetRequest{
			KeyId: keyId,
		})

	result, response, err := request.Execute()
	if err = after(
		span,
		response,
		err,
		"Exchanging public token with Plaid",
		"failed to exchange public token with Plaid",
	); err != nil {
		log.WithError(err).Errorf("failed to exchange public token with Plaid")
		return nil, err
	}

	webhook, err := NewWebhookVerificationKeyFromPlaid(result.Key)
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (p *Plaid) NewClientFromItemId(ctx context.Context, itemId string) (Client, error) {
	span := sentry.StartSpan(ctx, "Plaid - NewClientFromItemId")
	defer span.Finish()

	link, err := p.repo.GetLinkByItemId(span.Context(), itemId)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create client without link")
	}

	return p.newClient(span.Context(), link)
}

func (p *Plaid) NewClientFromLink(ctx context.Context, accountId uint64, linkId uint64) (Client, error) {
	panic("implement me")
}

func (p *Plaid) NewClient(ctx context.Context, link *models.Link, accessToken string) (Client, error) {
	return &PlaidClient{
		accountId:   link.AccountId,
		linkId:      link.LinkId,
		accessToken: accessToken,
		log: p.log.WithFields(logrus.Fields{
			"accountId": link.AccountId,
			"linkId":    link.LinkId,
		}),
		client: p.client,
	}, nil
}

func (p *Plaid) newClient(ctx context.Context, link *models.Link) (Client, error) {
	span := sentry.StartSpan(ctx, "Plaid - newClient")
	defer span.Finish()

	if link == nil {
		return nil, errors.New("cannot create client without link")
	}

	if link.PlaidLink == nil {
		return nil, errors.New("cannot create client without link")
	}

	accessToken, err := p.secret.GetAccessTokenForPlaidLinkId(span.Context(), link.AccountId, link.PlaidLink.ItemId)
	if err != nil {
		return nil, err
	}

	return p.NewClient(span.Context(), link, accessToken)
}

func (p *Plaid) Close() error {
	panic("implement me")
}
