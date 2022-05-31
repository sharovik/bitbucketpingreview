package bitbucketpingreview

import (
	"fmt"
	"github.com/sharovik/devbot/events/bitbucketpingreview/bitbucketpingreview_dto"
	"github.com/sharovik/devbot/internal/database"
	"github.com/sharovik/devbot/internal/helper"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sharovik/devbot/internal/log"

	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
)

const (
	//EventName the name of the event
	EventName = "bitbucketpingreview"

	//EventVersion the version of the event
	EventVersion = "1.0.3"

	helpMessage = "Ask me `ping reviewers for {PULL_REQUEST_1} {PULL_REQUEST_2} ... {PULL_REQUEST_N}` and I will ask the reviewers to review your pull-requests."

	pullRequestStringAnswer   = "I found the next pull-requests:\n"
	noPullRequestStringAnswer = `I can't find any pull-request in your message`

	pullRequestStateOpen   = "OPEN"

	pullRequestMinApprovals = 2

	pullRequestsRegex = `(?m)https:\/\/bitbucket.org\/(?P<workspace>\w+)\/(?P<repository_slug>[a-zA-Z0-9-_]+)\/pull-requests\/(?P<pull_request_id>\d+)`
)

//EventStruct the struct for the event object. It will be used for initialisation of the event in defined-events.go file.
type EventStruct struct {
	EventName string
}

//Event - object which is ready to use
var Event = EventStruct{
	EventName: EventName,
}

//ReceivedPullRequests struct for pull-requests list
type ReceivedPullRequests struct {
	Items []bitbucketpingreview_dto.PullRequest
}

//PullRequest the pull-request item
type PullRequest struct {
	ID             int64
	RepositorySlug string
	BranchName     string
	Workspace      string
	Title          string
	Description    string
}

var availableUsers dto.SlackResponseUsersList

//Execute method which is called by message processor
func (e EventStruct) Execute(message dto.BaseChatMessage) (dto.BaseChatMessage, error) {
	isHelpAnswerTriggered, err := helper.HelpMessageShouldBeTriggered(message.OriginalMessage.Text)
	if err != nil {
		log.Logger().Warn().Err(err).Msg("Something went wrong with help message parsing")
	}

	if isHelpAnswerTriggered {
		message.Text = helpMessage
		return message, nil
	}

	loadAvailableChannels()

	//First we need to find all the pull-requests in received message
	foundPullRequests := findAllPullRequestsInText(pullRequestsRegex, message.OriginalMessage.Text)

	//We prepare the text, where we define all the pull-requests which we found in the received message
	message.Text = receivedPullRequestsText(foundPullRequests)

	if len(foundPullRequests.Items) == 0 {
		return message, nil
	}

	toNotify := prepareListToNotify(foundPullRequests.Items, message.OriginalMessage.User)

	var notifiedReviewersList []string
	for user, toReview := range toNotify {
		userUUID := notifyReviewer(user, toReview, message.OriginalMessage.User)
		if userUUID == "" {
			continue
		}

		notifiedReviewersList = append(notifiedReviewersList, userUUID)
	}

	if len(notifiedReviewersList) == 0 {
		message.Text = "Unfortunately, I wasn't able to find the slack members for any of your pr"
		return message, nil
	}

	message.Text = fmt.Sprintf("I notified next persons: %s", strings.Join(notifiedReviewersList, ", "))
	return message, nil
}

func notifyReviewer(displayName string, toReview []string, fallbackChannelID string) (uuid string) {
	uuid = findChannelID(displayName)
	if uuid == "" {
		SendMessageToTheChannel(fallbackChannelID, fmt.Sprintf("Failed to notify user `%s`. It looks like his bitbucket profile is out of sync with Slack profile.", displayName))
		return
	}

	if uuid == fallbackChannelID {
		return ""
	}

	SendMessageToTheChannel(uuid, fmt.Sprintf("Hey <@%s>,\nUser <@%s> asked me to ping you regarding review for the next PR's: %s \nThanks!", uuid, fallbackChannelID, strings.Join(toReview, ",\n")))
	return fmt.Sprintf("<@%s>", uuid)
}

func findChannelID(displayName string) string {
	for _, user := range availableUsers.Members {
		if displayName == user.RealName {
			return user.ID
		}
	}

	return ""
}

func loadAvailableChannels() {
	if len(availableUsers.Members) > 0 {
		return
	}

	var err error
	availableUsers, _, err = container.C.MessageClient.GetUsersList()
	if err != nil {
		log.Logger().AddError(err).Msg("Failed to fetch the users list")
	}
}

func prepareListToNotify(items []bitbucketpingreview_dto.PullRequest, channelID string) (toNotify map[string][]string) {
	toNotify = make(map[string][]string)

	for _, pullRequest := range items {
		cleanPullRequestURL := fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", pullRequest.Workspace, pullRequest.RepositorySlug, pullRequest.ID)
		info, err := container.C.BibBucketClient.PullRequestInfo(pullRequest.Workspace, pullRequest.RepositorySlug, pullRequest.ID)
		if err != nil {
			SendMessageToTheChannel(channelID, fmt.Sprintf("Failed to fetch PR (%s) info. %s", cleanPullRequestURL, err))
			continue
		}

		replacer := strings.NewReplacer("\\", "")
		pullRequest.Title = info.Title
		pullRequest.BranchName = info.Source.Branch.Name
		pullRequest.RepositorySlug = info.Source.Repository.Name
		pullRequest.Description = replacer.Replace(info.Description)

		cleanPullRequestURL = fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", pullRequest.Workspace, pullRequest.RepositorySlug, pullRequest.ID)

		if isPRMerged(info) {
			SendMessageToTheChannel(channelID, fmt.Sprintf("PR #%d (%s) is already merged, I will not notify it's reviewers. Skipping...", pullRequest.ID, cleanPullRequestURL))
			continue
		}

		for _, reviewer := range info.Participants {
			if reviewer.Approved {
				continue
			}

			toNotify[reviewer.User.DisplayName] = deduplicate(cleanPullRequestURL, toNotify[reviewer.User.DisplayName])
		}

		if isReadyToMerge(info) {
			SendMessageToTheChannel(channelID, fmt.Sprintf("PR #%d (%s) - is ready to be merged!", pullRequest.ID, cleanPullRequestURL))
		}
	}

	return toNotify
}

func isReadyToMerge(info dto.BitBucketPullRequestInfoResponse) bool {
	var numberApproved = 0
	for _, participant := range info.Participants {
		if participant.Approved {
			numberApproved++
		}
	}

	return numberApproved > pullRequestMinApprovals
}

func deduplicate(item string, list []string) []string {
	for _, existingItem := range list {
		if item == existingItem {
			return list
		}
	}

	return append(list, item)
}

func findAllPullRequestsInText(regex string, subject string) ReceivedPullRequests {
	re, err := regexp.Compile(regex)

	if err != nil {
		log.Logger().AddError(err).Msg("Error during the Find Matches operation")
		return ReceivedPullRequests{}
	}

	matches := re.FindAllStringSubmatch(subject, -1)
	result := ReceivedPullRequests{}

	if len(matches) == 0 {
		return result
	}

	for _, id := range matches {
		if id[1] != "" {
			item := bitbucketpingreview_dto.PullRequest{}
			item.Workspace = id[1]
			item.RepositorySlug = id[2]
			item.ID, err = strconv.ParseInt(id[3], 10, 64)
			if err != nil {
				log.Logger().AddError(err).
					Interface("matches", matches).
					Msg("Error during pull-request ID parsing")
				return ReceivedPullRequests{}
			}

			result.Items = append(result.Items, item)
		}
	}

	return result
}

func receivedPullRequestsText(foundPullRequests ReceivedPullRequests) string {

	if len(foundPullRequests.Items) == 0 {
		return noPullRequestStringAnswer
	}

	var pullRequestsString = pullRequestStringAnswer
	for _, item := range foundPullRequests.Items {
		pullRequestsString = pullRequestsString + fmt.Sprintf("Pull-request #%d\n", item.ID)
	}

	return pullRequestsString
}

//Install method for installation of event
func (e EventStruct) Install() error {
	log.Logger().Debug().
		Str("event_name", EventName).
		Str("event_version", EventVersion).
		Msg("Triggered event installation")

	return container.C.Dictionary.InstallNewEventScenario(database.NewEventScenario{
		EventName:    EventName,
		EventVersion: EventVersion,
		Questions:    []database.Question{
			{
				Question:      "ping reviewers",
				Answer:        "One moment please",
				QuestionRegex: "(?i)(ping reviewers)",
				QuestionGroup: "",
			},
		},
	})
}

//Update for event update actions
func (e EventStruct) Update() error {
	return nil
}

func SendMessageToTheChannel(channel string, text string) {
	_, _, err := container.C.MessageClient.SendMessage(dto.SlackRequestChatPostMessage{
		Channel:           channel,
		Text:              text,
		AsUser:            true,
		Ts:                time.Time{},
		DictionaryMessage: dto.DictionaryMessage{},
		OriginalMessage:   dto.SlackResponseEventMessage{},
	})
	if err != nil {
		log.Logger().AddError(err).Str("text", text).Msg("Failed to send a message to the channel")
	}
}

func isPRMerged(info dto.BitBucketPullRequestInfoResponse) bool {
	if info.State == pullRequestStateOpen {
		return false
	}

	return true
}
