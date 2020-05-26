package alerts

import "github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"

type Delegate interface {
	NewCapturer() trackers.Capturer
}
