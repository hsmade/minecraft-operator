package webui

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type Api struct {
	Client client.Client
	Log    logr.Logger
}

// getServers gets the manifests for all Servers
func (a *Api) getServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var servers v1.ServerList
	err := a.Client.List(context.Background(), &servers)
	if err != nil {
		err = errors.Wrap(err, "failed to get Servers")
		//a.Log.Error(err, "failed to get Servers")
		returnError(err, w)
		return
	}

	a.Log.Info("got servers", "servers", servers.Items)
	err = json.NewEncoder(w).Encode(servers.Items)
	if err != nil {
		//a.Log.Error(err, "failed to serialize Servers")
		returnError(errors.Wrap(err, "failed to serialize Servers"), w)
		return
	}
}

// setServer sets the status for a Server (enabled: true/false)
func (a *Api) setServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	serverName, ok := r.URL.Query()["server"]
	if !ok || len(serverName[0]) < 1 {
		err := errors.New("missing server parameter")
		//a.Log.Error(err, "parsing parameters")
		returnError(err, w)
		return
	}

	enabledString, ok := r.URL.Query()["enabled"]
	if !ok || len(enabledString[0]) < 1 {
		err := errors.New("missing enabled parameter")
		//a.Log.Error(err, "parsing parameters")
		returnError(err, w)
		return
	}

	enabled, err := strconv.ParseBool(enabledString[0])
	if err != nil {
		err := errors.Wrap(err, "parsing enabled parameter to bool")
		//a.Log.Error(err, "")
		returnError(err, w)
		return
	}

	var server v1.Server
	err = a.Client.Get(context.Background(), types.NamespacedName{
		Name:      serverName[0],
		Namespace: "default", // FIXME: remove hard-coding
	}, &server)
	if err != nil {
		err := errors.Wrap(err, "retrieving Server object")
		a.Log.Error(err, "")
		returnError(err, w)
		return
	}

	server.Spec.Enabled = enabled
	err = a.Client.Update(context.Background(), &server)
	if err != nil {
		err := errors.Wrap(err, "updating Server object")
		a.Log.Error(err, "")
		returnError(err, w)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(nil)
}
