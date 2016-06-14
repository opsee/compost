package resolver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	etcd "github.com/coreos/etcd/client"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/opsee/basic/clients/hugs"
	"github.com/opsee/basic/schema"
	opsee "github.com/opsee/basic/service"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type checkCompostResponse struct {
	response interface{}
}

const (
	CheckResultTableName   = "check_results"
	CheckResponseTableName = "check_responses"

	RoutePath           = "/opsee.co/routes"
	MagicExecutionGroup = "127a7354-290e-11e6-b178-2bc1f6aefc14"
)

// ListChecks fetches Checks from Bartnet and CheckResults from Beavis
// concurrently, then zips them together. If the request to Beavis fails,
// then checks are returned without results.
func (c *Client) ListChecks(ctx context.Context, user *schema.User, checkId string) ([]*schema.Check, error) {
	var (
		responseChan = make(chan *checkCompostResponse, 2)
		checkMap     = make(map[string][]*schema.CheckResult)
		notifMap     = make(map[string][]*schema.Notification)
		wg           sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		var (
			results []*schema.CheckResult
			err     error
		)

		// a temporary "feature flag" to pull results from dynamo instead of beavis
		if user.Admin {

		}

		if checkId != "" {
			results, err = c.Beavis.ListResultsCheck(user, checkId)
		} else {
			results, err = c.Beavis.ListResults(user)
		}

		if err != nil {
			responseChan <- &checkCompostResponse{err}
		} else {
			responseChan <- &checkCompostResponse{results}
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		var (
			notifs []*hugs.Notification
			err    error
		)

		if checkId != "" {
			notifs, err = c.Hugs.ListNotificationsCheck(user, checkId)
		} else {
			notifs, err = c.Hugs.ListNotifications(user)
		}

		if err != nil {
			responseChan <- &checkCompostResponse{err}
		} else {
			responseChan <- &checkCompostResponse{notifs}
		}

		wg.Done()
	}()

	var (
		checks []*schema.Check
		err    error
	)

	if checkId != "" {
		check, err := c.Bartnet.GetCheck(user, checkId)
		if err != nil {
			log.WithError(err).Error("couldn't list checks from bartnet")
			return nil, err
		}

		checks = append(checks, check)
	} else {
		checks, err = c.Bartnet.ListChecks(user)
		if err != nil {
			log.WithError(err).Error("couldn't list checks from bartnet")
			return nil, err
		}
	}

	wg.Wait()
	close(responseChan)

	for resp := range responseChan {
		switch t := resp.response.(type) {
		case []*schema.CheckResult:
			for _, result := range t {
				for _, res := range result.Responses {
					if res.Reply == nil {
						if res.Response == nil {
							continue
						}

						any, err := opsee_types.UnmarshalAny(res.Response)
						if err != nil {
							log.WithError(err).Error("couldn't list results from beavis")
							return nil, err
						}

						switch reply := any.(type) {
						case *schema.HttpResponse:
							res.Reply = &schema.CheckResponse_HttpResponse{reply}
						case *schema.CloudWatchResponse:
							res.Reply = &schema.CheckResponse_CloudwatchResponse{reply}
						}
					}
				}

				if _, ok := checkMap[result.CheckId]; !ok {
					checkMap[result.CheckId] = []*schema.CheckResult{result}
				} else {
					checkMap[result.CheckId] = append(checkMap[result.CheckId], result)
				}
			}

		case []*hugs.Notification:
			for _, notif := range t {
				notifMap[notif.CheckId] = append(notifMap[notif.CheckId], &schema.Notification{Type: notif.Type, Value: notif.Value})
			}

		case error:
			log.WithError(t).Error("error composting checks")
		}
	}

	for _, check := range checks {
		check.Results = checkMap[check.Id]
		check.Notifications = notifMap[check.Id]

		if check.Spec == nil {
			if check.CheckSpec == nil {
				continue
			}

			any, err := opsee_types.UnmarshalAny(check.CheckSpec)
			if err != nil {
				log.WithError(err).Error("couldn't list checks from bartnet")
				return nil, err
			}

			switch spec := any.(type) {
			case *schema.HttpCheck:
				check.Spec = &schema.Check_HttpCheck{spec}
			case *schema.CloudWatchCheck:
				check.Spec = &schema.Check_CloudwatchCheck{spec}
			}
		}
	}

	return checks, nil
}

func (c *Client) UpsertChecks(ctx context.Context, user *schema.User, checksInput []interface{}) ([]*schema.Check, error) {
	notifs := make([]*hugs.NotificationRequest, 0, len(checksInput))
	checksResponse := make([]*schema.Check, len(checksInput))

	for i, checkInput := range checksInput {
		check, ok := checkInput.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("error decoding check input")
		}

		notifList, _ := check["notifications"].([]interface{})
		delete(check, "notifications")

		checkJson, err := json.Marshal(check)
		if err != nil {
			return nil, err
		}

		checkProto := &schema.Check{}
		err = jsonpb.Unmarshal(bytes.NewBuffer(checkJson), checkProto)
		if err != nil {
			return nil, err
		}

		var checkResponse *schema.Check

		if checkProto.Id == "" {
			checkResponse, err = c.Bartnet.CreateCheck(user, checkProto)
			if err != nil {
				return nil, err
			}
		} else {
			checkResponse, err = c.Bartnet.UpdateCheck(user, checkProto)
			if err != nil {
				return nil, err
			}
		}

		if notifList != nil {
			notif := &hugs.NotificationRequest{
				CheckId: checkResponse.Id,
			}

			for _, n := range notifList {
				nl, _ := n.(map[string]interface{})
				t, _ := nl["type"].(string)
				v, _ := nl["value"].(string)

				if t != "" && v != "" {
					// due to our crappy backend, we send a bulk request to hugs of all notifications...
					notif.Notifications = append(notif.Notifications, &hugs.Notification{
						Type:  t,
						Value: v,
					})

					// ... then we add each notification to the check object
					checkResponse.Notifications = append(checkResponse.Notifications, &schema.Notification{
						Type:  t,
						Value: v,
					})
				}
			}

			notifs = append(notifs, notif)
		}

		err = c.Hugs.CreateNotificationsMulti(user, notifs)
		if err != nil {
			return nil, err
		}

		checksResponse[i] = checkResponse
	}

	return checksResponse, nil
}

func (c *Client) DeleteChecks(ctx context.Context, user *schema.User, checksInput []interface{}) ([]string, error) {
	deleted := make([]string, 0, len(checksInput))
	for _, ci := range checksInput {
		id, ok := ci.(string)
		if !ok {
			return nil, fmt.Errorf("unable to decode check id")
		}

		err := c.Bartnet.DeleteCheck(user, id)
		if err != nil {
			continue
		}

		deleted = append(deleted, id)
	}

	return deleted, nil
}

func (c *Client) DeprecatedTestCheck(ctx context.Context, user *schema.User, checkInput map[string]interface{}) (*opsee.TestCheckResponse, error) {
	checkJson, err := json.Marshal(checkInput)
	if err != nil {
		return nil, err
	}

	checkProto := &schema.Check{}
	err = jsonpb.Unmarshal(bytes.NewBuffer(checkJson), checkProto)
	if err != nil {
		return nil, err
	}

	checkReponse, err := c.Bartnet.TestCheck(user, checkProto)
	if err != nil {
		return nil, err
	}

	for _, res := range checkReponse.Responses {
		if res.Reply == nil {
			if res.Response == nil {
				continue
			}

			any, err := opsee_types.UnmarshalAny(res.Response)
			if err != nil {
				return nil, err
			}

			switch reply := any.(type) {
			case *schema.HttpResponse:
				res.Reply = &schema.CheckResponse_HttpResponse{reply}
			case *schema.CloudWatchResponse:
				res.Reply = &schema.CheckResponse_CloudwatchResponse{reply}
			}
		}
	}

	return checkReponse, nil
}

func (c *Client) TestCheck(ctx context.Context, user *schema.User, checkInput map[string]interface{}) (*opsee.TestCheckResponse, error) {
	var (
		responses []*schema.CheckResponse
		exgroupId = user.CustomerId
	)

	checkJson, err := json.Marshal(checkInput)
	if err != nil {
		return nil, err
	}

	checkProto := &schema.Check{}
	err = jsonpb.Unmarshal(bytes.NewBuffer(checkJson), checkProto)
	if err != nil {
		return nil, err
	}

	if checkProto.Target == nil {
		return nil, fmt.Errorf("test check is missing target")
	}

	if checkProto.Target.Type == "external_host" {
		exgroupId = MagicExecutionGroup
	}

	// use customer id or execution group id ok!!
	response, err := c.EtcdKeys.Get(ctx, path.Join(RoutePath, exgroupId), &etcd.GetOptions{
		Recursive: true,
		Quorum:    true,
	})

	if len(response.Node.Nodes) == 0 {
		return nil, fmt.Errorf("no bastions found")
	}

	deadline := &opsee_types.Timestamp{}
	deadline.Scan(time.Now().UTC().Add(1 * time.Minute))

	for _, node := range response.Node.Nodes {
		services := make(map[string]interface{})

		err = json.Unmarshal([]byte(node.Value), &services)
		if err != nil {
			return nil, err
		}

		if checker, ok := services["checker"].(map[string]interface{}); ok {
			checkerHost, _ := checker["hostname"].(string)
			checkerPort, _ := checker["port"].(float64)
			addr := fmt.Sprintf("%s:%d", checkerHost, int(checkerPort))

			conn, err := grpc.Dial(
				addr,
				grpc.WithInsecure(),
				grpc.WithBlock(),
				grpc.WithTimeout(5*time.Second),
			)
			if err != nil {
				log.WithError(err).Errorf("coudln't contact bastion at: %s ... ignoring", addr)
				continue
			}

			resp, err := opsee.NewCheckerClient(conn).TestCheck(ctx, &opsee.TestCheckRequest{Deadline: deadline, Check: checkProto})
			if err != nil {
				log.WithError(err).Errorf("got error from bastion at: %s ... ignoring", addr)
				continue
			}

			responses = append(responses, resp.Responses...)
		}
	}

	return &opsee.TestCheckResponse{Responses: responses}, nil
}

func (c *Client) CheckResults(ctx context.Context, user *schema.User, checkId string) ([]*schema.CheckResult, error) {
	resp, err := c.Dynamo.Query(&dynamodb.QueryInput{
		TableName:              aws.String(CheckResultTableName),
		KeyConditionExpression: aws.String(fmt.Sprintf("check_id = %s", checkId)),
		ScanIndexForward:       aws.Bool(true),
		Select:                 aws.String("ALL_ATTRIBUTES"),
		Limit:                  aws.Int64(1),
	})
	if err != nil {
		return nil, err
	}

	results := make([]*schema.CheckResult, 0, len(resp.Items))
	for resultIdx, item := range resp.Items {
		resultId := item["result_id"]

		bastionResult := &schema.CheckResult{}
		if err := dynamodbattribute.UnmarshalMap(item, bastionResult); err != nil {
			return nil, err
		}

		grResp, err := c.Dynamo.Query(&dynamodb.QueryInput{
			TableName:              aws.String(CheckResponseTableName),
			KeyConditionExpression: aws.String(fmt.Sprintf("check_id = %s AND result_id = %s", checkId, resultId)),
			Select:                 aws.String("ALL_ATTRIBUTES"),
		})
		if err != nil {
			return nil, err
		}

		responses := make([]*schema.CheckResponse, 0, len(grResp.Items))
		for i, response := range grResp.Items {
			checkResponse := &schema.CheckResponse{}
			if err := dynamodbattribute.UnmarshalMap(response, checkResponse); err != nil {
				return nil, err
			}
			responses[i] = checkResponse
		}
		bastionResult.Responses = responses
		results[resultIdx] = bastionResult
	}

	return results, nil
}
