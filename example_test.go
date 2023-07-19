package cloudlog_test

import (
	"context"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/maddiesch/go-cloudlog"
	"golang.org/x/exp/slog"
)

func ExampleStructuredLogger() {
	awsConfig, _ := config.LoadDefaultConfig(context.TODO())
	cloudWatchClient := cloudwatchlogs.NewFromConfig(awsConfig)
	cloudLogPublisher := cloudlog.NewCloudWatchLogPublisher(cloudWatchClient, "log-group-name", "log-stream-name")

	cloud := cloudlog.NewWriter(cloudLogPublisher)
	defer cloud.Close()

	log := slog.New(slog.NewJSONHandler(io.MultiWriter(cloud, os.Stdout), nil))

	log.Info("Hello world!")
}
