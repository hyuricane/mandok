package compose

import (
	"context"
	"strconv"

	"github.com/docker/compose/v2/pkg/api"
)

type LogConsumer struct {
	logChan  chan string
	consumer func(logType, containerName, message string)
	api.LogConsumer
}

func (l *LogConsumer) Log(containerName, message string) {
	l.consumer("log", containerName, message)
}

func (l *LogConsumer) Err(containerName, message string) {
	l.consumer("err", containerName, message)
}

func (l *LogConsumer) Status(containerName, message string) {
	l.consumer("status", containerName, message)
}

func LogStream(projectDir, service string, tail int) (chan string, func(), error) {
	ctx := context.Background()
	return LogStreamWithContext(ctx, projectDir, service, tail)
}

func LogStreamWithContext(ctx context.Context, projectDir, service string, tail int) (chan string, func(), error) {
	project, err := LoadProject(ctx, projectDir)
	if err != nil {
		return nil, nil, err
	}
	services := []string{service}
	project.Project, err = project.Project.WithSelectedServices(services)
	if err != nil {
		return nil, nil, err
	}

	logChan := make(chan string, 100)
	cancelableCtx, cancel := context.WithCancel(ctx)

	go func(ctx context.Context) {
		// apiClient is github.com/docker/compose/v2/pkg/api#Compose
		apiClient := getAPI()
		err := apiClient.Logs(ctx, project.Project.Name, &LogConsumer{
			consumer: func(logType, containerName, message string) {
				logChan <- message
			},
		}, api.LogOptions{
			Project:  project.Project,
			Services: services,
			Tail:     strconv.Itoa(tail),
			Follow:   true,
		})
		if err != nil {
			logChan <- err.Error()
		}
		close(logChan)
	}(cancelableCtx)

	return logChan, cancel, nil
}
