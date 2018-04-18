package user

import (
	"context"
	"time"

	"github.com/hamakn/go_ddd_webapp/src/app/domain/user"
	"github.com/hamakn/go_ddd_webapp/src/app/infrastructure/concurrency"
	appDatastore "github.com/hamakn/go_ddd_webapp/src/app/infrastructure/datastore"
	"github.com/hamakn/go_ddd_webapp/src/app/infrastructure/db"
	"github.com/hamakn/go_ddd_webapp/src/app/infrastructure/fixture"
	"google.golang.org/appengine/datastore"
)

const (
	kind = "User"
)

type repository struct {
}

// NewRepository returns user.Repository
func NewRepository() user.Repository {
	return &repository{}
}

func (r *repository) GetAll(ctx context.Context) ([]*user.User, error) {
	users := []*user.User{}
	q := datastore.NewQuery(kind)

	_, err := db.GetAll(ctx, q, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*user.User, error) {
	u := &user.User{ID: id}
	err := db.Get(ctx, u)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, user.ErrNoSuchEntity
		}
		return nil, err
	}

	return u, nil
}

func (r *repository) Create(ctx context.Context, u *user.User) error {
	err := u.Validate()
	if err != nil {
		return user.ErrValidationFailed
	}

	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		err := concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				// check email uniqueness
				func() error {
					tctx := tctx
					if !canTakeUserEmail(tctx, u.Email) {
						return user.ErrEmailCannotTake
					}
					return nil
				},
				// check screen_name uniquness
				func() error {
					tctx := tctx
					if !canTakeUserScreenName(tctx, u.ScreenName) {
						return user.ErrScreenNameCannotTake
					}
					return nil
				},
				// Put user
				func() error {
					tctx := tctx
					return db.Put(tctx, u)
				},
			},
		)
		if err != nil {
			return err
		}

		// create userEmail and userScreenName after u.ID assigned
		userEmail := newUserEmail(u)
		userScreenName := newUserScreenName(u)

		err = concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				// take email
				func() error {
					tctx := tctx
					err = takeUserEmail(tctx, userEmail)
					if err != nil {
						return user.ErrEmailCannotTake
					}
					return nil
				},
				// take screen_name
				func() error {
					tctx := tctx
					err = takeUserScreenName(tctx, userScreenName)
					if err != nil {
						return user.ErrScreenNameCannotTake
					}
					return nil
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

func (r *repository) Update(ctx context.Context, u *user.User) error {
	err := u.Validate()
	if err != nil {
		return user.ErrValidationFailed
	}

	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		// get oldUser to get old email and screen name
		oldUser, err := r.GetByID(tctx, u.ID)
		if err != nil {
			return err
		}

		err = concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				// userEmail
				func() error {
					tctx := tctx
					if oldUser.Email != u.Email {
						return updateUserEmail(tctx, u, oldUser.Email)
					}
					return nil
				},
				// userScreenName
				func() error {
					tctx := tctx
					if oldUser.ScreenName != u.ScreenName {
						return updateUserScreenName(tctx, u, oldUser.ScreenName)
					}
					return nil
				},
				// Update user
				func() error {
					tctx := tctx

					// TODO: rollback if error occurred
					u.UpdatedAt = time.Now()

					return db.Put(tctx, u)
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

func (r *repository) Delete(ctx context.Context, u *user.User) error {
	return appDatastore.RunInTransaction(ctx, func(tctx context.Context) error {
		err := concurrency.ExecAllOrAbortOnError(
			tctx,
			[]func() error{
				// userEmail
				func() error {
					tctx := tctx
					return deleteUserEmail(tctx, u.Email, u.ID)
				},
				// userScreenName
				func() error {
					tctx := tctx
					return deleteUserScreenName(tctx, u.ScreenName, u.ID)
				},
				// delete user
				func() error {
					tctx := tctx
					// lock user
					txu, err := r.GetByID(tctx, u.ID)
					if err != nil {
						return err
					}
					return db.Delete(tctx, txu)
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

func (r *repository) CreateFixture(ctx context.Context) ([]*user.User, error) {
	users := []*user.User{}

	err := fixture.Load("users", &users)
	if err != nil {
		return nil, err
	}

	// NOTE: run in out of txn by datastore (25 entities) limit
	for _, u := range users {
		r.Create(ctx, u)
	}

	return users, nil
}
