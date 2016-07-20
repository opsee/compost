package schema

import (
	"errors"
)

var (
	errNoTeamName         = errors.New("missing team name")
	errNoTeamSubscription = errors.New("missing team subscription")
)

func (t *Team) Validate() error {
	if t.Name == "" {
		return errNoTeamName
	}

	if t.SubscriptionPlan == "" {
		return errNoTeamSubscription
	}

	return nil
}
