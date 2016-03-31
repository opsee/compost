package resolver

import (
	"github.com/opsee/basic/schema"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// ListChecks fetches Checks from Bartnet and CheckResults from Beavis
// concurrently, then zips them together. If the request to Beavis fails,
// then checks are returned without results.
func (c *client) ListChecks(ctx context.Context, user *schema.User) ([]*schema.Check, error) {
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
			if _, ok := checkMap[result.CheckId]; !ok {
				checkMap[result.CheckId] = []*schema.CheckResult{result}
			} else {
				checkMap[result.CheckId] = append(checkMap[result.CheckId], result)
			}
		}

		for _, check := range checks {
			check.Results = checkMap[check.Id]
			if check.Spec == nil {
				any, err := schema.UnmarshalAny(check.CheckSpec)
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
