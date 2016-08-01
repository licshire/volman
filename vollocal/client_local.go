package vollocal

import (
	"errors"
	"time"

	"github.com/tedsuo/ifrit"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/metric"
	"code.cloudfoundry.org/voldriver"
	"github.com/cloudfoundry-incubator/volman"
)

const (
	volmanMountErrorsCounter   = metric.Counter("VolmanMountErrors")
	volmanMountDuration        = metric.Duration("VolmanMountDuration")
	volmanUnmountErrorsCounter = metric.Counter("VolmanUnmountErrors")
	volmanUnmountDuration      = metric.Duration("VolmanUnmountDuration")
)

type DriverConfig struct {
	DriverPaths  []string
	SyncInterval time.Duration
}

func NewDriverConfig() DriverConfig {
	return DriverConfig{
		SyncInterval: time.Second * 30,
	}
}

type localClient struct {
	driverRegistry DriverRegistry
	clock          clock.Clock
}

func NewServer(logger lager.Logger, config DriverConfig) (volman.Manager, ifrit.Runner) {
	clock := clock.NewClock()
	registry := NewDriverRegistry()
	return NewLocalClient(logger, registry, clock), NewDriverSyncer(logger, registry, config.DriverPaths, config.SyncInterval, clock).Runner()
}

func NewLocalClient(logger lager.Logger, registry DriverRegistry, clock clock.Clock) volman.Manager {
	return &localClient{
		driverRegistry: registry,
		clock:          clock,
	}
}

func (client *localClient) ListDrivers(logger lager.Logger) (volman.ListDriversResponse, error) {
	logger = logger.Session("list-drivers")
	logger.Info("start")
	defer logger.Info("end")

	var infoResponses []volman.InfoResponse
	drivers := client.driverRegistry.Drivers()

	for name, _ := range drivers {
		infoResponses = append(infoResponses, volman.InfoResponse{Name: name})
	}

	logger.Debug("listing-drivers", lager.Data{"drivers": infoResponses})
	return volman.ListDriversResponse{infoResponses}, nil
}

func (client *localClient) Mount(logger lager.Logger, driverId string, volumeId string, config map[string]interface{}) (volman.MountResponse, error) {
	logger = logger.Session("mount")
	logger.Info("start")
	defer logger.Info("end")

	mountStart := client.clock.Now()

	defer func() {
		err := volmanMountDuration.Send(time.Since(mountStart))
		if err != nil {
			logger.Error("failed-to-send-volman-mount-duration-metric", err)
		}
	}()

	logger.Debug("driver-mounting-volume", lager.Data{"driverId": driverId, "volumeId": volumeId})

	driver, found := client.driverRegistry.Driver(driverId)
	if !found {
		err := errors.New("Driver '" + driverId + "' not found in list of known drivers")
		logger.Error("mount-driver-lookup-error", err)
		volmanMountErrorsCounter.Increment()
		return volman.MountResponse{}, err
	}

	err := client.create(logger, driverId, volumeId, config)
	if err != nil {
		volmanMountErrorsCounter.Increment()
		return volman.MountResponse{}, err
	}

	mountRequest := voldriver.MountRequest{Name: volumeId}
	logger.Debug("calling-driver-with-mount-request", lager.Data{"driverId": driverId, "mountRequest": mountRequest})
	mountResponse := driver.Mount(logger, mountRequest)
	logger.Debug("response-from-driver", lager.Data{"response": mountResponse})
	if mountResponse.Err != "" {
		volmanMountErrorsCounter.Increment()
		return volman.MountResponse{}, errors.New(mountResponse.Err)
	}

	return volman.MountResponse{mountResponse.Mountpoint}, nil
}

func (client *localClient) Unmount(logger lager.Logger, driverId string, volumeName string) error {
	logger = logger.Session("unmount")
	logger.Info("start")
	defer logger.Info("end")
	logger.Debug("unmounting-volume", lager.Data{"volumeName": volumeName})

	unmountStart := client.clock.Now()

	defer func() {
		err := volmanUnmountDuration.Send(time.Since(unmountStart))
		if err != nil {
			logger.Error("failed-to-send-volman-unmount-duration-metric", err)
		}
	}()

	driver, found := client.driverRegistry.Driver(driverId)
	if !found {
		err := errors.New("Driver '" + driverId + "' not found in list of known drivers")
		logger.Error("mount-driver-lookup-error", err)
		volmanUnmountErrorsCounter.Increment()
		return err
	}

	if response := driver.Unmount(logger, voldriver.UnmountRequest{Name: volumeName}); response.Err != "" {
		err := errors.New(response.Err)
		logger.Error("unmount-failed", err)
		volmanUnmountErrorsCounter.Increment()
		return err
	}

	return nil
}

func (client *localClient) create(logger lager.Logger, driverId string, volumeName string, opts map[string]interface{}) error {
	logger = logger.Session("create")
	logger.Info("start")
	defer logger.Info("end")
	driver, found := client.driverRegistry.Driver(driverId)
	if !found {
		err := errors.New("Driver '" + driverId + "' not found in list of known drivers")
		logger.Error("mount-driver-lookup-error", err)
		return err
	}

	logger.Debug("creating-volume", lager.Data{"volumeName": volumeName, "driverId": driverId, "opts": opts})
	response := driver.Create(logger, voldriver.CreateRequest{Name: volumeName, Opts: opts})
	if response.Err != "" {
		return errors.New(response.Err)
	}
	return nil
}
