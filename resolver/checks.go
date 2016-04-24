package resolver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/opsee/basic/clients/hugs"
	"github.com/opsee/basic/schema"
	opsee_types "github.com/opsee/protobuf/opseeproto/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// ListChecks fetches Checks from Bartnet and CheckResults from Beavis
// concurrently, then zips them together. If the request to Beavis fails,
// then checks are returned without results.
func (c *Client) ListChecks(ctx context.Context, user *schema.User) ([]*schema.Check, error) {
	var (
		resultChan = make(chan []*schema.CheckResult)
		errChan    = make(chan error)
		checkMap   = make(map[string][]*schema.CheckResult)
	)

	go func() {
		results, err := c.Beavis.ListResults(user)
		if err != nil {
			errChan <- err
			return
		}

		resultChan <- results
	}()

	checks, err := c.Bartnet.ListChecks(user)
	if err != nil {
		log.WithError(err).Error("couldn't list checks from bartnet")
		return nil, err
	}

	select {
	case results := <-resultChan:
		for _, result := range results {
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

		for _, check := range checks {
			check.Results = checkMap[check.Id]
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

	case err = <-errChan:
		log.WithError(err).Error("couldn't list results from beavis, we're returning checks anyhow")
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
