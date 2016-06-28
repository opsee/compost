package schema

func (r *CheckResult) Targets() []*Target {
	var targets []*Target
	if r != nil {
		for _, resp := range r.Responses {
			targets = append(targets, resp.Target)
		}
	}
	return targets
}

func (r *CheckResult) filterResponses(passing bool) []*CheckResponse {
	responses := []*CheckResponse{}
	if r != nil {
		for _, resp := range r.Responses {
			if resp.Passing == passing {
				responses = append(responses, resp)
			}
		}
	}
	return responses
}

func (r *CheckResult) countResponses(passing bool) int {
	count := 0
	if r != nil {
		for _, resp := range r.Responses {
			if resp.Passing == passing {
				count += 1
			}
		}
	}
	return count
}

func (r *CheckResult) PassingResponses() []*CheckResponse {
	return r.filterResponses(true)
}

func (r *CheckResult) FailingResponses() []*CheckResponse {
	return r.filterResponses(false)
}

func (r *CheckResult) FailingCount() int {
	return r.countResponses(false)
}

func (r *CheckResult) PassingCount() int {
	return r.countResponses(true)
}
