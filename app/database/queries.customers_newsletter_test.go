package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomerNewsletter(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	customer, err := Db.CreateCustomer(&Customer{Email: "newsletter@example.com"})
	require.NoError(t, err)
	require.False(t, customer.Newsletter)
	require.Nil(t, customer.NewsletterAt)

	require.NoError(t, Db.SubscribeCustomerToNewsletter(customer.ID))
	subscribed, err := Db.GetCustomerByID(customer.ID)
	require.NoError(t, err)
	require.True(t, subscribed.Newsletter)
	require.NotNil(t, subscribed.NewsletterAt)

	// Subscribing again keeps the original opt-in date
	require.NoError(t, Db.SubscribeCustomerToNewsletter(customer.ID))
	again, err := Db.GetCustomerByID(customer.ID)
	require.NoError(t, err)
	require.True(t, subscribed.NewsletterAt.Equal(*again.NewsletterAt))

	// Saving the customer unchanged keeps the opt-in date as well
	unchanged, err := Db.UpdateCustomer(again)
	require.NoError(t, err)
	require.True(t, subscribed.NewsletterAt.Equal(*unchanged.NewsletterAt))

	// Unsubscribing clears the opt-in date
	unchanged.Newsletter = false
	unsubscribed, err := Db.UpdateCustomer(unchanged)
	require.NoError(t, err)
	require.False(t, unsubscribed.Newsletter)
	require.Nil(t, unsubscribed.NewsletterAt)
}
