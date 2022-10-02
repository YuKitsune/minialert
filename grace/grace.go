package grace

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type shutdownHook func() error

func runHooks(logger logrus.FieldLogger, shutdownHooks []shutdownHook) {
	for _, hook := range shutdownHooks {
		err := hook()
		if err != nil {
			logger.WithError(err).Errorf("Error executing shutdown hook: %s", err)
		}
	}
}

func WaitForShutdownSignal(logger logrus.FieldLogger, shutdownHooks ...shutdownHook) {
	defer runHooks(logger, shutdownHooks)

	shutdownChan := getShutdownSignalChan()
	waitForSignal(logger, shutdownChan, make(chan error, 1))
}

func WaitForShutdownSignalOrError(logger logrus.FieldLogger, errorChan chan error, shutdownHooks ...shutdownHook) {
	defer runHooks(logger, shutdownHooks)

	shutdownChan := getShutdownSignalChan()
	waitForSignal(logger, shutdownChan, errorChan)
}

func waitForSignal(logger logrus.FieldLogger, shutdownSignalChan chan os.Signal, errorChan chan error) {
	select {
	case sig := <-shutdownSignalChan:
		handleShutdownSignal(logger, sig)
		break

	case err := <-errorChan:
		ExitFromError(logger, err)
		break
	}
}

func handleShutdownSignal(logger logrus.FieldLogger, sig os.Signal) {
	if sig == syscall.SIGTERM || sig == syscall.SIGQUIT || sig == syscall.SIGINT || sig == os.Kill {
		logger.Infof("🛑 Signal caught: %s", sig.String())
		go func() {
			select {
			// exit if graceful shutdown not finished in 60 sec.
			case <-time.After(time.Second * 60):
				logger.Fatalln("⏰ Graceful shutdown timed out")
			}
		}()
		return
	}

	logger.Infof("🤷 Shutting down, unknown signal caught: %s", sig.String())
	return
}

func getShutdownSignalChan() chan os.Signal {
	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGINT,
		syscall.SIGKILL,
	)

	return shutdownSignalChan
}

func ExitFromError(logger logrus.FieldLogger, err error) {
	logger.WithError(err).Fatalf(err.Error())
}

func FExitFromError(writer io.Writer, err error) {
	_, _ = fmt.Fprintf(writer, "error: %s", err.Error())
	os.Exit(1)
}
