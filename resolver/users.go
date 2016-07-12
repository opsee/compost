package resolver

import (
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	log "github.com/opsee/logrus"
	"golang.org/x/net/context"
)

func (c *Client) ListCustomers(ctx context.Context, req *opsee.ListUsersRequest) (*opsee.ListCustomersResponse, error) {
	log.Debug("list users request")

	resp, err := c.Cats.ListUsers(ctx, req)
	if err != nil {
		log.WithError(err).Error("error listing users from cats")
		return nil, err
	}

	// as a shim until we have a cats endpoint for listing customers, unique the customers with a map
	customerIdMap := make(map[string][]*schema.User)
	for _, user := range resp.Users {
		_, ok := customerIdMap[user.CustomerId]
		if !ok {
			customerIdMap[user.CustomerId] = make([]*schema.User, 0)
		}
		customerIdMap[user.CustomerId] = append(customerIdMap[user.CustomerId], user)
	}

	customerIds := make([]string, 0, len(customerIdMap))
	for customerId, _ := range customerIdMap {
		customerIds = append(customerIds, customerId)
	}

	stateResp, err := c.Keelhaul.ListBastionStates(ctx, &opsee.ListBastionStatesRequest{CustomerIds: customerIds})
	if err != nil {
		log.WithError(err).Error("error listing users from keelhaul")
		return nil, err
	}

	bastionStates := make(map[string][]*schema.BastionState)
	for _, bastionState := range stateResp.BastionStates {
		_, ok := bastionStates[bastionState.CustomerId]
		if !ok {
			bastionStates[bastionState.CustomerId] = make([]*schema.BastionState, 0)
		}
		bastionStates[bastionState.CustomerId] = append(bastionStates[bastionState.CustomerId], bastionState)
	}

	// this stuff is also a crummy shim for no customer listing endpoint
	customers := make([]*schema.Customer, 0, len(customerIds))
	for cid, userlist := range customerIdMap {
		customers = append(customers, &schema.Customer{
			Id:            cid,
			Name:          userlist[0].Name,
			CreatedAt:     userlist[0].CreatedAt,
			UpdatedAt:     userlist[0].UpdatedAt,
			Users:         userlist[:1],
			BastionStates: bastionStates[cid],
		})
	}

	return &opsee.ListCustomersResponse{
		Customers: customers,
		Page:      resp.Page,
		PerPage:   resp.PerPage,
		Total:     resp.Total,
	}, nil
}

func (c *Client) GetUser(ctx context.Context, req *opsee.GetUserRequest) (*opsee.GetUserResponse, error) {
	log.WithFields(log.Fields{
		"customer_id": req.CustomerId,
		"id":          req.Id,
		"email":       req.Email,
	}).Debug("get user request")

	resp, err := c.Cats.GetUser(ctx, req)
	if err != nil {
		log.WithError(err).Error("error getting user from cats")
		return nil, err
	}

	return resp, nil
}

func (c *Client) PutUser(ctx context.Context, req *opsee.UpdateUserRequest) (*schema.User, error) {
	log.WithFields(log.Fields{
		"customer_id": req.User.CustomerId,
		"id":          req.User.Id,
		"email":       req.User.Email,
	}).Debug("update user request")

	resp, err := c.Cats.UpdateUser(ctx, req)
	if err != nil {
		log.WithError(err).Error("error updating user")
		return nil, err
	}

	return resp.User, nil
}

func (c *Client) InviteUser(ctx context.Context, req *opsee.InviteUserRequest) (*schema.User, error) {
	log.WithFields(log.Fields{
		"customer_id": req.Requestor.CustomerId,
		"email":       req.Email,
		"perms":       req.Perms.HighFlags(),
	}).Debug("invite user request")

	resp, err := c.Cats.InviteUser(ctx, req)
	if err != nil {
		log.WithError(err).Error("error inviting user")
		return nil, err
	}
	if resp.Invite == nil {
		return &schema.User{}, nil
	}

	return &schema.User{
		Id:         resp.Invite.Id,
		Email:      resp.Invite.Email,
		CustomerId: resp.Invite.CustomerId,
		Perms:      resp.Invite.Perms,
	}, nil
}
