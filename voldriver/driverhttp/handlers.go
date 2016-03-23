package driverhttp

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	cf_http_handlers "github.com/cloudfoundry-incubator/cf_http/handlers"
	"github.com/cloudfoundry-incubator/volman/voldriver"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func NewHandler(logger lager.Logger, client voldriver.Driver) (http.Handler, error) {
	logger = logger.Session("server")
	logger.Info("start")
	defer logger.Info("end")
	var handlers = rata.Handlers{

		"get": http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			logger = logger.Session("get-mount")
			logger.Info("start")
			defer logger.Info("end")

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				logger.Error("failed-reading-get-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.MountResponse{Err: err.Error()})
				return
			}

			var getRequest voldriver.GetRequest
			if err = json.Unmarshal(body, &getRequest); err != nil {
				logger.Error("failed-unmarshalling-get-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.GetResponse{Err: err.Error()})
				return
			}

			getResponse := client.Get(logger, getRequest)
			if getResponse.Err != "" {
				logger.Error("failed-getting-volume", err, lager.Data{"volume": getRequest.Name})
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, getResponse)
				return
			}

			cf_http_handlers.WriteJSONResponse(w, http.StatusOK, getResponse)
		}),

		"mount": http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			logger = logger.Session("handle-mount")
			logger.Info("start")
			defer logger.Info("end")

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				logger.Error("failed-reading-mount-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.MountResponse{Err: err.Error()})
				return
			}

			var mountRequest voldriver.MountRequest
			if err = json.Unmarshal(body, &mountRequest); err != nil {
				logger.Error("failed-unmarshalling-mount-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.MountResponse{Err: err.Error()})
				return
			}

			mountResponse := client.Mount(logger, mountRequest)
			if mountResponse.Err != "" {
				logger.Error("failed-mounting-volume", errors.New(mountResponse.Err), lager.Data{"volume": mountRequest.Name})
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, mountResponse)
				return
			}

			cf_http_handlers.WriteJSONResponse(w, http.StatusOK, mountResponse)
		}),

		"unmount": http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			logger = logger.Session("handle-unmount")
			logger.Info("start")
			defer logger.Info("end")

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				logger.Error("failed-reading-unmount-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.ErrorResponse{Err: err.Error()})
				return
			}

			var unmountRequest voldriver.UnmountRequest
			if err = json.Unmarshal(body, &unmountRequest); err != nil {
				logger.Error("failed-unmarshalling-unmount-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.ErrorResponse{Err: err.Error()})
				return
			}

			unmountResponse := client.Unmount(logger, unmountRequest)
			if unmountResponse.Err != "" {
				logger.Error("failed-unmount-volume", errors.New(unmountResponse.Err), lager.Data{"volume": unmountRequest.Name})
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, unmountResponse)
				return
			}

			cf_http_handlers.WriteJSONResponse(w, http.StatusOK, unmountResponse)
		}),

		"create": http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			logger = logger.Session("handle-create")
			logger.Info("start")
			defer logger.Info("end")

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				logger.Error("failed-reading-create-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.ErrorResponse{Err: err.Error()})
				return
			}

			var createRequest voldriver.CreateRequest
			if err = json.Unmarshal(body, &createRequest); err != nil {
				logger.Error("failed-unmarshalling-create-request-body", err)
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, voldriver.ErrorResponse{Err: err.Error()})
				return
			}

			createResponse := client.Create(logger, createRequest)
			if createResponse.Err != "" {
				logger.Error("failed-creating-volume", errors.New(createResponse.Err))
				cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, createResponse)
				return
			}

			cf_http_handlers.WriteJSONResponse(w, http.StatusOK, createResponse)
		}),
	}

	return rata.NewRouter(voldriver.Routes, handlers)
}