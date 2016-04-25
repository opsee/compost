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
	"sync"
)

type checkCompostResponse struct {
	response interface{}
}

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

		case []*hugs.Notification:
			for _, notif := range t {
				notifMap[notif.CheckId] = append(notifMap[notif.CheckId], &schema.Notification{Type: notif.Type, Value: notif.Value})
			}

			for _, check := range checks {
				check.Notifications = notifMap[check.Id]
			}

		case error:
			log.WithError(err).Error("error composting checks")
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
