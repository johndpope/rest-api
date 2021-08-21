package platypus

import (
	"github.com/jarcoal/httpmock"
	"github.com/monetr/rest-api/pkg/internal/mock_plaid"
	"github.com/plaid/plaid-go/plaid"
	"testing"
)

func TestPlaidClient_GetAccount(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.Deactivate()

		account := mock_plaid.BankAccountFixture(t)

		mock_plaid.MockGetAccounts(t, []plaid.AccountBase{
			account,
		})


	})
}
