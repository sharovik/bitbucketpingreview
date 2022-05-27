# Bitbucket ping reviewers
Sometimes happens, that your Pull-request for some reason miss the review. In that case you always need to go to each user and ping them manually. I introduced a new event, which can help you with that. Just type `ping revewers {PULL_REQUESTS_LIST}` and devbot will ping the reviewers in slack to review these PR's.

## Installation guide
To install it please run 
``` 
make build-installation-script && scripts/install/run --event_alias=bitbucketpingreview
```

## Usage
Write in PM or tag the bot user with this message
```
ping reviewers {PULL_REQUEST_1} {PULL_REQUEST_2} ... {PULL_REQUEST_N}
```
