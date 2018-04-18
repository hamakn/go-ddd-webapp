package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hamakn/go_ddd_webapp/src/app/domain/user"
	"github.com/hamakn/go_ddd_webapp/src/app/infrastructure/concurrency"
	appDatastore "github.com/hamakn/go_ddd_webapp/src/app/infrastructure/datastore"
	"google.golang.org/appengine/datastore"
)

// userEmail Entity for email uniqueness constraints
type userEmail struct {
	Email     string
	UserID    int64
	CreatedAt time.Time
}

func newUserEmail(u *user.User) *userEmail {
	now := time.Now()
	return &userEmail{
		Email:     u.Email,
		UserID:    u.ID,
		CreatedAt: now,
	}
}

func userEmailKey(ctx context.Context, email string) *datastore.Key {
	return datastore.NewKey(ctx, "UserEmail", userEmailKeyString(email), 0, nil)
}

// userEmailKeyString is downcased email
func userEmailKeyString(email string) string {
	return strings.ToLower(email)
}

func canTakeUserEmail(ctx context.Context, email string) bool {
	k := userEmailKey(ctx, email)
	err := datastore.Get(ctx, k, &userEmail{})
	if err != nil && err == datastore.ErrNoSuchEntity {
		return true
	}
	return false
}

func takeUserEmail(ctx context.Context, u *userEmail) error {
	k := userEmailKey(ctx, u.Email)
	_, err := datastore.Put(ctx, k, u)
	return err
}

func updateUserEmail(ctx context.Context, u *user.User, oldEmail string) error {
	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		// lock new userEmail
		if !canTakeUserEmail(tctx, u.Email) {
			return user.ErrEmailCannotTake
		}

		err := concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				func() error {
					tctx := tctx
					return deleteUserEmail(tctx, oldEmail, u.ID)
				},
				func() error {
					tctx := tctx
					return takeUserEmail(tctx, newUserEmail(u))
				},
			},
		)
		if err != nil {
			return err
		}

		return nil
	},
		true, // XG
	)
}

func deleteUserEmail(ctx context.Context, email string, userID int64) error {
	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		// lock old userEmail
		old := userEmail{}
		err := datastore.Get(tctx, userEmailKey(tctx, email), &old)
		if err != nil {
			return err
		}
		// checking owner of old userEmail
		if old.UserID != userID {
			return errors.New("app-infra-db-user-email: oldEmail is not specified user's")
		}

		err = datastore.Delete(tctx, userEmailKey(tctx, email))
		if err != nil {
			return err
		}

		return nil
	},
		false,
	)
}
