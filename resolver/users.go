package resolver

import (
	"time"

	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (c *Client) ListCustomers(ctx context.Context, req *opsee.ListUsersRequest) (*opsee.ListCustomersResponse, error) {
	log.Info("list users request")

	resp, err := c.Vape.ListUsers(ctx, req)
	if err != nil {
		log.WithError(err).Error("error listing users from vape")
		return nil, err
	}

	// as a shim until we have a vape endpoint for listing customers, unique the customers with a map
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
	}).Info("get user request")

	resp, err := c.Vape.GetUser(ctx, req)
	if err != nil {
		log.WithError(err).Error("error getting user from vape")
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetTeam(ctx context.Context, user *schema.User) (*schema.Team, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("get team request")

	// FIXME(mark)
	date1 := &opsee_types.Timestamp{}
	date2 := &opsee_types.Timestamp{}
	date1.Scan(time.Now().UTC().Add(-2 * 24 * 30 * time.Hour))
	date2.Scan(time.Now().UTC().Add(-1 * 24 * 30 * time.Hour))

	resp := &schema.Team{
		Id:           "deadbeef",
		Name:         "SeaOrg",
		Subscription: "basic",
		CreditCardInfo: &schema.CreditCardInfo{
			ExpMonth: int32(4),
			ExpYear:  int32(1992),
			Last4:    "6969",
			Brand:    "visa",
		},
		Users: []*schema.User{
			{
				Id:    7,
				Name:  "Tom Cruise",
				Email: "tom@cruise.com",
			},
			{
				Id:    8,
				Name:  "Isaac Johannsen",
				Email: "izs@cruise.com",
			},
		},
		Invoices: []*schema.Invoice{
			{
				Date:   date1,
				Amount: int32(6699),
			},
			{
				Date:   date2,
				Amount: int32(6699),
			},
		},
	}

	return resp, nil
}

func (c *Client) PutTeam(ctx context.Context, user *schema.User, teamInput map[string]interface{}) (*schema.Team, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("put team request")

	// FIXME(mark)
	date1 := &opsee_types.Timestamp{}
	date2 := &opsee_types.Timestamp{}
	date1.Scan(time.Now().UTC().Add(-2 * 24 * 30 * time.Hour))
	date2.Scan(time.Now().UTC().Add(-1 * 24 * 30 * time.Hour))

	resp := &schema.Team{
		Id:           "deadbeef",
		Name:         "SeaOrg",
		Subscription: "basic",
		CreditCardInfo: &schema.CreditCardInfo{
			ExpMonth: int32(4),
			ExpYear:  int32(1992),
			Last4:    "6969",
			Brand:    "visa",
		},
		Users: []*schema.User{
			{
				Id:    7,
				Name:  "Tom Cruise",
				Email: "tom@cruise.com",
			},
			{
				Id:    8,
				Name:  "Isaac Johannsen",
				Email: "izs@cruise.com",
			},
		},
		Invoices: []*schema.Invoice{
			{
				Date:   date1,
				Amount: int32(6699),
			},
			{
				Date:   date2,
				Amount: int32(6699),
			},
		},
	}

	return resp, nil
}

func (c *Client) PutUser(ctx context.Context, user *schema.User, userInput map[string]interface{}) (*schema.User, error) {
	log.WithFields(log.Fields{
		"customer_id": user.CustomerId,
		"email":       user.Email,
	}).Info("put team request")

	// FIXME(mark)
	resp := &schema.User{
		Id:    7,
		Name:  "Tom Cruise",
		Email: "tom@cruise.com",
	}

	return resp, nil
}
