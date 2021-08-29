package webui

import (
	"embed"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/fs"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed assets
var webroot embed.FS

func Run(addr string, kClient client.Client, Log logr.Logger) error {
	sub, err := fs.Sub(webroot, "assets")
	if err != nil {
		return errors.Wrap(err, "getting FS to assets/")
	}

	api := Api{Client: kClient, Log: Log.WithName("api")}
	http.HandleFunc("/api/server/logs", api.getServerLogs)
	http.HandleFunc("/api/server/command", api.postServerCommand)
	http.HandleFunc("/api/server", api.setServer)
	http.HandleFunc("/api/servers", api.getServers)
	http.Handle("/", http.FileServer(http.FS(sub)))
	return http.ListenAndServe(addr, nil)
}
