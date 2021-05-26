package stripe_helper

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v72"
	stripe_client "github.com/stripe/stripe-go/v72/client"
	"net/http"
	"time"
)

type Stripe interface {
	AttachPaymentMethod(ctx context.Context, paymentMethodId, customerId string) (*stripe.PaymentMethod, error)
	GetPricesById(ctx context.Context, stripePriceIds []string) ([]stripe.Price, error)
	GetProductsById(ctx context.Context, stripeProductIds []string) ([]stripe.Product, error)
	CreateSubscription(ctx context.Context, subscription stripe.SubscriptionParams) (*stripe.Subscription, error)
	CreateCustomer(ctx context.Context, customer stripe.CustomerParams) (*stripe.Customer, error)
	UpdateCustomer(ctx context.Context, id string, customer stripe.CustomerParams) (*stripe.Customer, error)
}

var (
	_ Stripe = &stripeBase{}
)

type stripeBase struct {
	log    *logrus.Entry
	client *stripe_client.API
}

func NewStripeHelper(log *logrus.Entry, apiKey string) Stripe {
	return &stripeBase{
		log: log,
		client: stripe_client.New(apiKey, stripe.NewBackends(&http.Client{
			Timeout: time.Second * 30,
		})),
	}
}

func (s *stripeBase) GetPricesById(ctx context.Context, stripePriceIds []string) ([]stripe.Price, error) {
	span := sentry.StartSpan(ctx, "Stripe - GetPricesById")
	defer span.Finish()

	prices := make([]stripe.Price, len(stripePriceIds))
	for i, stripePriceId := range stripePriceIds {
		price, err := s.GetPriceById(span.Context(), stripePriceId)
		if err != nil {
			return nil, err
		}

		prices[i] = *price
	}

	return prices, nil
}

func (s *stripeBase) GetPriceById(ctx context.Context, id string) (*stripe.Price, error) {
	span := sentry.StartSpan(ctx, "Stripe - GetPriceById")
	defer span.Finish()

	log := s.log.WithField("stripePriceId", id)

	result, err := s.client.Prices.Get(id, &stripe.PriceParams{})
	if err != nil {
		log.WithError(err).Error("failed to retrieve stripe price")
		return nil, errors.Wrap(err, "failed to retrieve stripe price")
	}

	return result, nil
}

func (s *stripeBase) GetProductsById(ctx context.Context, stripeProductIds []string) ([]stripe.Product, error) {
	span := sentry.StartSpan(ctx, "Stripe - GetProductsById")
	defer span.Finish()

	productIds := make([]*string, len(stripeProductIds))
	for i := range stripeProductIds {
		productIds[i] = &stripeProductIds[i]
	}

	productIterator := s.client.Products.List(&stripe.ProductListParams{
		IDs: productIds,
	})

	products := make([]stripe.Product, 0)
	for {
		if err := productIterator.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve stripe products")
		}

		if !productIterator.Next() {
			break
		}

		if product := productIterator.Product(); product != nil {
			products = append(products, *product)
		}
	}

	return products, nil
}

func (s *stripeBase) AttachPaymentMethod(ctx context.Context, paymentMethodId, customerId string) (*stripe.PaymentMethod, error) {
	span := sentry.StartSpan(ctx, "Stripe - AttachPaymentMethod")
	defer span.Finish()

	result, err := s.client.PaymentMethods.Attach(paymentMethodId, &stripe.PaymentMethodAttachParams{
		Customer: &customerId,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to attach payment method")
	}

	return result, nil
}

func (s *stripeBase) CreateSubscription(ctx context.Context, subscription stripe.SubscriptionParams) (*stripe.Subscription, error) {
	span := sentry.StartSpan(ctx, "Stripe - CreateSubscription")
	defer span.Finish()

	result, err := s.client.Subscriptions.New(&subscription)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create subscription")
	}

	return result, nil
}

func (s *stripeBase) CreateCustomer(ctx context.Context, customer stripe.CustomerParams) (*stripe.Customer, error) {
	span := sentry.StartSpan(ctx, "Stripe - CreateCustomer")
	defer span.Finish()

	result, err := s.client.Customers.New(&customer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create customer")
	}

	return result, nil
}

func (s *stripeBase) UpdateCustomer(ctx context.Context, id string, customer stripe.CustomerParams) (*stripe.Customer, error) {
	span := sentry.StartSpan(ctx, "Stripe - UpdateCustomer")
	defer span.Finish()

	result, err := s.client.Customers.Update(id, &customer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update customer")
	}

	return result, nil
}