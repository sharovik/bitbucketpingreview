# Bitbucket ping reviewers
Sometimes happens, that your Pull-request for some reason miss the review. In that case you always need to go to each user and ping them manually. I introduced a new event for [devbot application](https://github.com/sharovik/devbot), which can help you with that. Just type `ping revewers {PULL_REQUESTS_LIST}` and [devbot](https://github.com/sharovik/devbot) will ping the reviewers in slack to review these PR's.

### Clone into devbot project
```
git clone git@github.com:sharovik/bitbucketpingreview.git events/bitbucketpingreview
```

### Install it into your devbot project
1. clone this repository into `events/` folder of your devbot project. Please make sure to use `bitbucketpingreview` folder name for this event
2. add this event into `defined-events.go` file to the defined events map object
``` 
import (
    //...
	"github.com/sharovik/devbot/events/bitbucketpingreview"
)

// DefinedEvents variable contains the list of events, which will be installed/used by the devbot
var DefinedEvents = []event.DefinedEventInterface{
	//...
	bitbucketpingreview.Event,
}
```

## Usage
Write in PM or tag the bot user with this message
```
ping reviewers {PULL_REQUEST_1} {PULL_REQUEST_2} ... {PULL_REQUEST_N}
```
