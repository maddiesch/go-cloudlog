package cloudlog

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudWatchLogClient interface {
	PutLogEvents(context.Context, *cloudwatchlogs.PutLogEventsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
}

func NewCloudWatchLogPublisher(client CloudWatchLogClient, group, stream string) *CloudWatchLogPublisher {
	return &CloudWatchLogPublisher{
		Client:        client,
		LogGroupName:  group,
		LogStreamName: stream,
	}
}

type CloudWatchLogPublisher struct {
	Client        CloudWatchLogClient
	LogGroupName  string
	LogStreamName string
}

func (c *CloudWatchLogPublisher) PublishLogEvents(ctx context.Context, events []LogEvent) error {
	if len(events) == 0 {
		return nil
	}

	var logEvents []types.InputLogEvent

	for _, event := range events {
		logEvents = append(logEvents, types.InputLogEvent{
			Message:   &event.Message,
			Timestamp: &event.Timestamp,
		})
	}

	_, err := c.Client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  &c.LogGroupName,
		LogStreamName: &c.LogStreamName,
		LogEvents:     logEvents,
	})

	return err
}

var _ CloudLogPublisher = (*CloudWatchLogPublisher)(nil)
