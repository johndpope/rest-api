package repository

import (
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/harderthanitneedstobe/rest-api/v0/pkg/models"
	"github.com/pkg/errors"
)

var (
	_ UnauthenticatedRepository = &unauthenticatedRepo{}
)

type unauthenticatedRepo struct {
	txn *pg.Tx
}

func (u *unauthenticatedRepo) CreateLogin(
	email, hashedPassword string, isEnabled bool,
) (*models.Login, error) {
	login := &models.Login{
		Email:        strings.ToLower(email),
		PasswordHash: hashedPassword,
		IsEnabled:    isEnabled,
	}
	count, err := u.txn.Model(login).
		Where(`"login"."email" = ?`, email).
		Count()
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify if email is unique")
	}

	if count != 0 {
		return nil, errors.Errorf("a login with the same email already exists")
	}

	_, err = u.txn.Model(login).Insert(login)
	return login, errors.Wrap(err, "failed to create login")
}

func (u *unauthenticatedRepo) CreateAccount(timezone *time.Location) (*models.Account, error) {
	account := &models.Account{
		Timezone: timezone.String(),
	}
	_, err := u.txn.Model(account).Insert(account)
	return account, errors.Wrap(err, "failed to create account")
}

func (u *unauthenticatedRepo) CreateUser(loginId, accountId uint64, firstName, lastName string) (*models.User, error) {
	user := &models.User{
		LoginId:          loginId,
		AccountId:        accountId,
		StripeCustomerId: "",
		FirstName:        firstName,
		LastName:         lastName,
	}
	_, err := u.txn.Model(user).Insert(user)
	return user, errors.Wrap(err, "failed to create user")
}

func (u *unauthenticatedRepo) CreateRegistration(loginId uint64) (*models.Registration, error) {
	now := time.Now().UTC()
	registration := &models.Registration{
		RegistrationId: "", // Will be generated when we insert it.
		LoginId:        loginId,
		DateCreated:    now,
		DateExpires:    now.Add(7 * 24 * time.Hour), // Expires in 7 days.
	}

	_, err := u.txn.Model(registration).Insert(registration)
	return nil, errors.Wrap(err, "failed to create registration")
}
