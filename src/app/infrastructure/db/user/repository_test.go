package user

import (
	"testing"
	"time"

	"github.com/hamakn/go_ddd_webapp/src/app/domain/user"
	"github.com/hamakn/go_ddd_webapp/src/app/infrastructure/environments"
	"github.com/hamakn/go_ddd_webapp/src/app/internal"
	"github.com/stretchr/testify/require"
	"google.golang.org/appengine/aetest"
)

func TestCreate(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	defer done()
	require.Nil(t, err)

	f := user.NewFactory()
	r := NewRepository()
	_, err = r.CreateFixture(ctx)
	require.Nil(t, err)

	testCases := []struct {
		email      string
		screenName string
		age        int
		hasError   bool
		err        error
	}{
		{
			// taken email (other case)
			"FOO@hamakn.test",
			"new_name",
			25,
			true,
			user.ErrEmailCannotTake,
		},
		{
			// taken screen name (other case)
			"new@hamakn.test",
			"FOO",
			26,
			true,
			user.ErrScreenNameCannotTake,
		},
		{
			// validation failed
			"bad_email",
			"new_name",
			27,
			true,
			user.ErrValidationFailed,
		},
		{
			// ok
			"new@hamakn.test",
			"new",
			17,
			false,
			nil,
		},
	}

	for _, testCase := range testCases {
		u := f.Create(testCase.email, testCase.screenName, testCase.age)
		err := r.Create(ctx, u)

		if testCase.hasError {
			require.NotNil(t, err)
			require.Equal(t, testCase.err, err)

		} else {
			require.Nil(t, err)
		}
	}
}

func TestUpdate(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	defer done()
	require.Nil(t, err)

	r := NewRepository()
	_, err = r.CreateFixture(ctx)
	require.Nil(t, err)

	now := time.Now()

	testCases := []struct {
		userID     int64
		email      string
		screenName string
		hasError   bool
		err        error
	}{
		{
			// NG1
			1,
			"BAR@hamakn.test",
			"new",
			true,
			user.ErrEmailCannotTake,
		},
		{
			// NG2
			1,
			"new@hamakn.test",
			"BAR",
			true,
			user.ErrScreenNameCannotTake,
		},
		{
			// NG3: validation failed
			1,
			"new@hamakn.test",
			"ｂａｄｎａｍｅ",
			true,
			user.ErrValidationFailed,
		},
		{
			// OK1
			1,
			"new@hamakn.test",
			"new",
			false,
			nil,
		},
		{
			// OK2
			// depends previous test case
			2,
			"foo@hamakn.test",
			"foo",
			false,
			nil,
		},
	}

	for _, testCase := range testCases {
		u, err := r.GetByID(ctx, testCase.userID)
		require.Nil(t, err)

		oldEmail := u.Email
		oldScreenName := u.ScreenName
		newAge := 99
		u.Email = testCase.email
		u.ScreenName = testCase.screenName
		u.Age = newAge
		err = r.Update(ctx, u)

		if testCase.hasError {
			require.NotNil(t, err)
			require.Equal(t, testCase.err, err)

		} else {
			require.Nil(t, err)

			u, err := r.GetByID(ctx, testCase.userID)
			require.Nil(t, err)
			require.Equal(t, testCase.email, u.Email)
			require.Equal(t, testCase.screenName, u.ScreenName)
			require.Equal(t, newAge, u.Age)
			require.Equal(t, true, u.UpdatedAt.After(now))

			require.Equal(t, true, canTakeUserEmail(ctx, oldEmail))
			require.Equal(t, false, canTakeUserEmail(ctx, testCase.email))

			require.Equal(t, true, canTakeUserScreenName(ctx, oldScreenName))
			require.Equal(t, false, canTakeUserScreenName(ctx, testCase.screenName))
		}
	}
}

func TestDelete(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	defer done()
	require.Nil(t, err)

	r := NewRepository()
	_, err = r.CreateFixture(ctx)
	require.Nil(t, err)

	testCases := []struct {
		userID   int64
		hasError bool
	}{
		{
			// OK
			1,
			false,
		},
	}

	for _, testCase := range testCases {
		u, err := r.GetByID(ctx, testCase.userID)
		require.Nil(t, err)

		deleteEmail := u.Email
		deleteScreenName := u.ScreenName
		err = r.Delete(ctx, u)

		if testCase.hasError {
			require.NotNil(t, err)

		} else {
			require.Nil(t, err)

			_, err := r.GetByID(ctx, testCase.userID)
			require.Equal(t, user.ErrNoSuchEntity, err)

			require.Equal(t, true, canTakeUserEmail(ctx, deleteEmail))
			require.Equal(t, true, canTakeUserScreenName(ctx, deleteScreenName))
		}
	}
}

func TestCreateFixture(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	defer done()
	require.Nil(t, err)

	r := NewRepository()
	users, err := r.CreateFixture(ctx)

	require.Nil(t, err)
	require.Equal(t, 2, len(users))
}

func init() {
	internal.MockEnvironments(&environments.Environments{})
}
