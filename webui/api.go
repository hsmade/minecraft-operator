package webui

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/pkg/errors"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Api struct {
	Client client.Client
	Log    logr.Logger
}

func (a *Api) getServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var servers v1.ServerList
	err := a.Client.List(context.Background(), &servers)
	if err != nil {
		a.Log.Error(err, "failed to get Servers")
		returnError(errors.Wrap(err, "failed to get Servers"), w)
		return
	}

	a.Log.Info("got servers", "servers", servers.Items)
	err = json.NewEncoder(w).Encode(servers.Items)
	if err != nil {
		a.Log.Error(err, "failed to serialize Servers")
		returnError(errors.Wrap(err, "failed to serialize Servers"), w)
		return
	}
}
