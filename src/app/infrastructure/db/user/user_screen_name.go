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

// userScreenName Entity for screen_name uniqueness constraints
type userScreenName struct {
	ScreenName string
	UserID     int64
	CreatedAt  time.Time
}

func newUserScreenName(u *user.User) *userScreenName {
	now := time.Now()
	return &userScreenName{
		ScreenName: u.ScreenName,
		UserID:     u.ID,
		CreatedAt:  now,
	}
}

func userScreenNameKey(ctx context.Context, screenName string) *datastore.Key {
	return datastore.NewKey(ctx, "UserScreenName", userScreenNameKeyString(screenName), 0, nil)
}

// userScreenNameKeyString is downcased email
func userScreenNameKeyString(screenName string) string {
	return strings.ToLower(screenName)
}

func canTakeUserScreenName(ctx context.Context, screenName string) bool {
	k := userScreenNameKey(ctx, screenName)
	err := datastore.Get(ctx, k, &userScreenName{})
	if err != nil && err == datastore.ErrNoSuchEntity {
		return true
	}
	return false
}

func takeUserScreenName(ctx context.Context, u *userScreenName) error {
	k := userScreenNameKey(ctx, u.ScreenName)
	_, err := datastore.Put(ctx, k, u)
	return err
}

func updateUserScreenName(ctx context.Context, u *user.User, oldScreenName string) error {
	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		// lock new userScreenName
		if !canTakeUserScreenName(tctx, u.ScreenName) {
			return user.ErrScreenNameCannotTake
		}

		err := concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				func() error {
					tctx := tctx
					return deleteUserScreenName(tctx, oldScreenName, u.ID)
				},
				func() error {
					tctx := tctx
					return takeUserScreenName(tctx, newUserScreenName(u))
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

func deleteUserScreenName(ctx context.Context, screenName string, userID int64) error {
	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		// lock old userScreenName
		old := userScreenName{}
		err := datastore.Get(tctx, userScreenNameKey(tctx, screenName), &old)
		if err != nil {
			return err
		}
		// checking owner of old userScreenName
		if old.UserID != userID {
			return errors.New("app-infra-db-user-screen_name: oldScreenName is not specified user's")
		}

		err = datastore.Delete(tctx, userScreenNameKey(tctx, screenName))
		if err != nil {
			return err
		}

		return nil
	},
		false,
	)
}
