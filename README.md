# CloudLog

Write log messages to CloudWatch Logs using a standard interface.

```golang
awsConfig, _ := config.LoadDefaultConfig(context.TODO())
cloudWatchClient := cloudwatchlogs.NewFromConfig(awsConfig)
cloudLogPublisher := cloudlog.NewCloudWatchLogPublisher(cloudWatchClient, "log-group-name", "log-stream-name")

cloud := cloudlog.NewWriter(cloudLogPublisher)
defer cloud.Close()

log := slog.New(slog.NewJSONHandler(io.MultiWriter(cloud, os.Stdout), nil))

log.Info("Hello world!")
```
